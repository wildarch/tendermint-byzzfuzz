package byzzfuzz

import (
	"log"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

const maxHeight = 3

func ByzzFuzzExpectNewRound(sp *common.SystemParams) *testlib.TestCase {
	// TODO: We should fix the fault node idx
	isolatedValidator := 0

	drops := []MessageDrop{
		// ROUND 0
		// Drops everything from isolatedValidator
		{round: 0, from: isolatedValidator, to: 0},
		{round: 0, from: isolatedValidator, to: 1},
		{round: 0, from: isolatedValidator, to: 2},
		{round: 0, from: isolatedValidator, to: 3},
		// Drops everything to isolatedValidator
		{round: 0, from: 0, to: isolatedValidator},
		{round: 0, from: 1, to: isolatedValidator},
		{round: 0, from: 2, to: isolatedValidator},
		{round: 0, from: 3, to: isolatedValidator},
	}

	allNodes := []int{0, 1, 2, 3}
	corruptions := []MessageCorruption{
		{round: 0, to: &allNodes, corruption: ChangeVoteToNil},
		{round: 1, to: &allNodes, corruption: ChangeVoteToNil},
	}

	return ByzzFuzzInst(sp, drops, corruptions)
}

func dropMessageLoudly() testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		message, ok := util.GetMessageFromEvent(e, c)
		if ok {
			from := -1
			to := -1
			for i, r := range c.Replicas.Iter() {
				if r.ID == message.From {
					from = i
				}
				if r.ID == message.To {
					to = i
				}
			}
			totalRounds, ok := c.Vars.GetInt(totalRoundsKey(e.Replica))
			if !ok {
				totalRounds = 0
			}
			log.Printf("Dropping message (from=%d, to=%d, round=%d)", from, to, totalRounds)
		} else {
			log.Printf("Dropping message!")
		}
		return []*types.Message{}
	}
}

// TODO: Use ReplicaIDs
func isMessageFrom(replicaIdx int) testlib.Condition {
	return func(e *types.Event, ctx *testlib.Context) bool {
		message, ok := ctx.GetMessage(e)
		if !ok {
			return false
		}
		return message.From == ctx.Replicas.Iter()[replicaIdx].ID
	}
}

func isMessageTo(replicaIdx int) testlib.Condition {
	return func(e *types.Event, ctx *testlib.Context) bool {
		message, ok := ctx.GetMessage(e)
		if !ok {
			return false
		}
		return message.To == ctx.Replicas.Iter()[replicaIdx].ID
	}
}

func IsMessageToOneOf(replicaIdxs []int) testlib.Condition {
	return func(e *types.Event, ctx *testlib.Context) bool {
		message, ok := ctx.GetMessage(e)
		if !ok {
			return false
		}
		for replicaIdx := range replicaIdxs {
			if message.To == ctx.Replicas.Iter()[replicaIdx].ID {
				return true
			}
		}
		return false
	}
}

func expectNewRound(sp *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	// We want replicas in partition "h" to move to round 1
	init.On(
		common.IsNewHeightRoundFromPart("h", 1, 1),
		testlib.SuccessStateLabel,
	)
	newRound := init.On(
		testlib.Count("round1ToH").Geq(sp.F+1),
		"newRoundMessagesDelivered",
	).On(
		common.IsNewHeightRoundFromPart("h", 1, 1),
		"NewRound",
	)
	newRound.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)
	newRound.On(
		common.IsCommit(),
		testlib.SuccessStateLabel,
	)

	init.On(
		common.IsCommit(),
		testlib.FailStateLabel,
	)

	filters := testlib.NewFilterSet()
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsVoteFromFaulty()),
		).Then(
			common.ChangeVoteToNil(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageReceive().
				And(common.IsMessageFromRound(1)).
				And(common.IsMessageToPart("h")).
				And(
					common.IsMessageType(util.Proposal).
						Or(common.IsMessageType(util.Prevote)).
						Or(common.IsMessageType(util.Precommit)),
				),
		).Then(
			testlib.Count("round1ToH").Incr(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageToPart("h")).
				And(common.IsMessageType(util.Prevote).Or(common.IsMessageType(util.Precommit))),
		).Then(
			testlib.DropMessage(),
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsVoteFromPart("h")),
		).Then(
			testlib.DropMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"ExpectNewRound",
		1*time.Minute,
		sm,
		filters,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
