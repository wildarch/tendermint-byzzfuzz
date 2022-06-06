package byzzfuzz

import (
	"byzzfuzz/byzzfuzz/spec"
	"byzzfuzz/liveness"
	"fmt"
	"strings"
	"time"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

type MessageDrop struct {
	Step      int `json:"step"`
	Partition Partition
}

func (d *MessageDrop) MessageType() util.MessageType {
	switch d.Step % 3 {
	case 0:
		return util.Proposal
	case 1:
		return util.Prevote
	case 2:
		return util.Precommit
	default:
		panic("impossible")
	}
}

func (d *MessageDrop) Round() int {
	return d.Step / 3
}

type MessageCorruption struct {
	Step       int            `json:"step"`
	From       int            `json:"from"`
	To         []int          `json:"to"`
	Corruption CorruptionType `json:"corruption_type"`
}

func (c *MessageCorruption) MessageType() util.MessageType {
	switch c.Step % 3 {
	case 0:
		return util.Proposal
	case 1:
		return util.Prevote
	case 2:
		return util.Precommit
	default:
		panic("impossible")
	}
}

func (c *MessageCorruption) Round() int {
	return c.Step / 3
}

type CorruptionType int

const (
	ChangeProposalToNil CorruptionType = iota
	ChangeVoteToNil
	ChangeVoteRound
	Omit
)

var ProposalCorruptionTypes = []CorruptionType{
	ChangeProposalToNil,
	Omit,
}

var VoteCorruptionTypes = []CorruptionType{
	ChangeVoteToNil,
	ChangeVoteRound,
	Omit,
}

const maxHeight = 3

const DiffCommitsLabel = "diff-commits"

func ByzzFuzzInst(sp *common.SystemParams, drops []MessageDrop, corruptions []MessageCorruption, timeout time.Duration) (*testlib.TestCase, chan spec.Event) {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.On(spec.DiffCommits, DiffCommitsLabel)
	init.On(common.HeightReached(maxHeight), testlib.SuccessStateLabel)

	filters := testlib.NewFilterSet()
	filters.AddFilter(trackTotalRounds)
	specEventCh := make(chan spec.Event, 10000)
	filters.AddFilter(spec.Log(specEventCh))

	filters.AddFilter(logConsensusMessages)

	for _, drop := range drops {
		filters.AddFilter(
			testlib.If(
				testlib.IsMessageSend().
					And(isMessageFromTotalRound(drop.Round())).
					And(common.IsMessageType(drop.MessageType())).
					And(FromToIsolated(drop.Partition)),
			).Then(dropMessageLoudly),
		)
	}

	for _, corruption := range corruptions {
		filters.AddFilter(
			testlib.If(testlib.IsMessageSend().
				And(isMessageFromTotalRound(corruption.Round())).
				And(common.IsMessageType(corruption.MessageType())).
				And(common.IsMessageFromPart(nodeLabel(corruption.From))).
				And(IsMessageToOneOf(corruption.To)),
			).Then(corruption.Action()),
		)
	}

	filters.AddFilter(testlib.If(common.HeightReached(maxHeight).And(sm.InState(testlib.SuccessStateLabel))).Then(endTest))

	testcase := testlib.NewTestCase("ByzzFuzzInst", timeout+liveness.ExtraTimeout, sm, filters)
	testcase.SetupFunc(common.Setup(sp, labelNodes, liveness.SetupLivenessTimer(timeout)))

	return testcase, specEventCh
}

func endTest(e *types.Event, c *testlib.Context) []*types.Message {
	c.EndTestCase()
	return []*types.Message{}
}

func nodeLabel(idx int) string {
	return fmt.Sprintf("node%d", idx)
}

func labelNodes(c *testlib.Context) {
	parts := make([]*util.Part, len(c.Replicas.Iter()))
	for i, replica := range c.Replicas.Iter() {
		replicaSet := util.NewReplicaSet()
		replicaSet.Add(replica)
		parts[i] = &util.Part{
			ReplicaSet: replicaSet,
			Label:      nodeLabel(i),
		}
	}
	partition := util.NewPartition(parts...)
	c.Vars.Set("partition", partition)
	c.Logger().With(log.LogParams{
		"partition": partition.String(),
	}).Info("Partitioned replicas")
}

func IsMessageToOneOf(replicaIdxs []int) testlib.Condition {
	cond := testlib.Condition(func(e *types.Event, c *testlib.Context) bool { return false })
	for _, replicaIdx := range replicaIdxs {
		cond = cond.Or(common.IsMessageToPart(nodeLabel(replicaIdx)))
	}
	return cond
}

func dropMessageLoudly(e *types.Event, c *testlib.Context) (message []*types.Message) {
	m, ok := util.GetMessageFromEvent(e, c)
	if ok {
		c.Logger().With(log.LogParams{
			"from":   getPartLabel(c, m.From),
			"to":     getPartLabel(c, m.To),
			"type":   m.Type,
			"height": m.Height(),
			"round":  m.Round(),
		}).Debug("Dropping message")
	} else {
		c.Logger().Warn("Dropping message with unknown height/round")
	}
	return
}

func getPartLabel(ctx *testlib.Context, id types.ReplicaID) string {
	partitionR, ok := ctx.Vars.Get("partition")
	if !ok {
		panic("No partition found")
	}
	partition := partitionR.(*util.Partition)
	for _, p := range partition.Parts {
		if strings.HasPrefix(p.Label, "node") && p.Contains(id) {
			return p.Label
		}
	}
	panic("Replica not found")
}
