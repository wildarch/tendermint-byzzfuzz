package byzzfuzz

import (
	"byzzfuzz/byzzfuzz/spec"
	"byzzfuzz/liveness"
	"math/rand"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

func BaselineTestCase(
	sp *common.SystemParams,
	dropPercent int,
	corruptPercent int) *testlib.TestCase {

	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.On(spec.DiffCommits, DiffCommitsLabel)
	init.On(common.IsCommit().And(liveness.IsTestFinished), testlib.SuccessStateLabel)

	filters := testlib.NewFilterSet()
	filters.AddFilter(testlib.If(sm.InState(testlib.SuccessStateLabel)).Then(endTest))
	filters.AddFilter(trackTotalRounds)

	filters.AddFilter(logConsensusMessages)

	rand.Seed(time.Now().UnixNano())

	filters.AddFilter(testlib.If(
		testlib.IsMessageSend().And(
			randomlyPick(dropPercent))).Then(dropMessageLoudly))
	filters.AddFilter(testlib.If(
		testlib.IsMessageSend().And(
			common.IsMessageFromPart("node0").And(
				randomlyPick(corruptPercent)))).Then(garbleMessage()))

	/*
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
					And(isMessageOfTotalRound(corruption.Round())).
					And(common.IsMessageType(corruption.MessageType())).
					And(common.IsMessageFromPart(nodeLabel(corruption.From))).
					And(IsMessageToOneOf(corruption.To)),
				).Then(corruption.Action()),
			)
		}
	*/

	testcase := testlib.NewTestCase("Baseline", 2*time.Minute, sm, filters)
	testcase.SetupFunc(common.Setup(sp, labelNodes, liveness.SetupLivenessTimer(time.Minute)))

	return testcase
}

func randomlyPick(pct int) testlib.Condition {
	return func(e *types.Event, c *testlib.Context) bool {
		if liveness.IsTestFinished(e, c) {
			return false
		}
		n := rand.Intn(100)
		if n < pct {
			return true
		}
		return false
	}
}

func garbleMessage() testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		c.Logger().Info("Corrupt (bitwise)")
		m, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(m)
		if !ok {
			return []*types.Message{m}
		}

		// Select a byte to corrupt
		byteIndex := rand.Intn(len(tMsg.MsgB))
		origByte := tMsg.MsgB[byteIndex]

		// Select a bit to corrupt
		bitIndex := rand.Intn(8)
		// Flip
		corByte := origByte ^ (1 << bitIndex)
		tMsg.MsgB[byteIndex] = corByte

		tMsg.Data = nil
		newMsg, err := tMsg.Marshal()
		if err != nil {
			return []*types.Message{m}
		}
		return []*types.Message{c.NewMessage(m, newMsg)}
	}
}
