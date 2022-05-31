package byzzfuzz

import (
	"fmt"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

func ExpectNewRound(sp *common.SystemParams) *testlib.TestCase {
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
			func(e *types.Event, c *testlib.Context) (ms []*types.Message) {
				m, ok := c.GetMessage(e)
				if !ok {
					return []*types.Message{}
				}
				message, ok := util.GetParsedMessage(m)
				if !ok {
					return []*types.Message{}
				}
				if ok {
					fmt.Printf("ChangeVoteToNil H=%d/R=%d %s %s -> %s [%s]\n", message.Height(), message.Round(), message.Type, message.From, message.To, e.Replica)
				}
				return
			},
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
			func(e *types.Event, c *testlib.Context) (ms []*types.Message) {
				fmt.Printf("+1 to H (dropped)\n")
				return
			},
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
			func(e *types.Event, c *testlib.Context) (ms []*types.Message) {
				m, ok := c.GetMessage(e)
				if !ok {
					return []*types.Message{}
				}
				message, ok := util.GetParsedMessage(m)
				if !ok {
					return []*types.Message{}
				}
				if ok {
					fmt.Printf("Drop H=%d/R=%d %s %s -> %s\n", message.Height(), message.Round(), message.Type, message.From, message.To)
				}
				return
			},
		),
	)
	filters.AddFilter(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsVoteFromPart("h")),
		).Then(
			testlib.DropMessage(),
			func(e *types.Event, c *testlib.Context) (ms []*types.Message) {
				m, ok := c.GetMessage(e)
				if !ok {
					return []*types.Message{}
				}
				message, ok := util.GetParsedMessage(m)
				if !ok {
					return []*types.Message{}
				}
				if ok {
					fmt.Printf("Drop H=%d/R=%d %s %s -> %s\n", message.Height(), message.Round(), message.Type, message.From, message.To)
				}
				return
			},
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
