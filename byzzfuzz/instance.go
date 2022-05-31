package byzzfuzz

import (
	"byzzfuzz/byzzfuzz/spec"
	"fmt"
	"time"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

type MessageDrop struct {
	Step int `json:"step"`
	From int `json:"from"`
	To   int `json:"to"`
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
)

var ProposalCorruptionTypes = []CorruptionType{
	ChangeProposalToNil,
}

var VoteCorruptionTypes = []CorruptionType{
	ChangeVoteToNil,
	ChangeVoteRound,
}

const maxHeight = 3

func ByzzFuzzInst(sp *common.SystemParams, drops []MessageDrop, corruptions []MessageCorruption, timeout time.Duration) (*testlib.TestCase, chan spec.Event) {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.MarkSuccess()
	init.On(spec.DiffCommits, testlib.FailStateLabel)

	filters := testlib.NewFilterSet()
	filters.AddFilter(trackTotalRounds)
	//filters.AddFilter(spec.TrackCurrentHeightRound)
	specEventCh := make(chan spec.Event, 10000)
	filters.AddFilter(spec.Log(specEventCh))

	for _, drop := range drops {
		filters.AddFilter(
			testlib.If(
				testlib.IsMessageSend().
					And(isMessageFromTotalRound(drop.Round())).
					And(common.IsMessageType(drop.MessageType())).
					And(common.IsMessageFromPart(nodeLabel(drop.From))).
					And(common.IsMessageToPart(nodeLabel(drop.To))),
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

	filters.AddFilter(testlib.If(common.HeightReached(maxHeight)).Then(endTest))

	testcase := testlib.NewTestCase("ByzzFuzzInst", timeout, sm, filters)
	testcase.SetupFunc(common.Setup(sp, labelNodes))

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
			"from":   m.From,
			"to":     m.To,
			"type":   m.Type,
			"height": m.Height(),
			"round":  m.Round(),
		}).Debug("Dropping message")
	} else {
		c.Logger().Warn("Dropping message with unknown height/round")
	}
	return
}
