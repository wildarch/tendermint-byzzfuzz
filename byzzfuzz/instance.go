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
	Round int `json:"round"`
	From  int `json:"from"`
	To    int `json:"to"`
}

type MessageCorruption struct {
	Round      int            `json:"round"`
	From       int            `json:"from"`
	To         []int          `json:"to"`
	Corruption CorruptionType `json:"corruption_type"`
}

type CorruptionType int

const (
	ChangeProposalToNil CorruptionType = iota
	ChangeVoteToNil
	ChangeVoteRound
)

var CorruptionTypes = []CorruptionType{
	ChangeProposalToNil,
	ChangeVoteToNil,
	ChangeVoteRound,
}

const maxHeight = 3

func ByzzFuzzInst(sp *common.SystemParams, drops []MessageDrop, corruptions []MessageCorruption, timeout time.Duration) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	maxHeightReached := init.On(common.HeightReached(maxHeight), "maxHeightReached")
	maxHeightReached.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)
	// TODO: Check if we expect consensus to be possible based on number of network faults
	maxHeightReached.On(
		common.IsCommit(),
		testlib.SuccessStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(trackTotalRounds)
	filters.AddFilter(spec.RecordHighestRoundNumberReceived)
	// Testing
	filters.AddFilter(testlib.If(spec.SendsMessageWithTooLowRound).Then(
		func(e *types.Event, ctx *testlib.Context) []*types.Message {
			ctx.Abort()
			return nil
		},
	))

	for _, drop := range drops {
		filters.AddFilter(
			testlib.If(
				testlib.IsMessageSend().
					And(isMessageFromTotalRound(drop.Round)).
					And(common.IsMessageFromPart(nodeLabel(drop.From))).
					And(common.IsMessageToPart(nodeLabel(drop.To))),
			).Then(testlib.DropMessage()),
		)
	}

	for _, corruption := range corruptions {
		action := actionForCorruption(corruption.Corruption)

		filters.AddFilter(
			testlib.If(testlib.IsMessageSend().
				And(isMessageFromTotalRound(corruption.Round)).
				And(common.IsMessageFromPart(nodeLabel(corruption.From))).
				And(IsMessageToOneOf(corruption.To)),
			).Then(action),
		)
	}

	testcase := testlib.NewTestCase("ByzzFuzzInst", timeout, sm, filters)
	testcase.SetupFunc(common.Setup(sp, labelNodes))

	return testcase
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
