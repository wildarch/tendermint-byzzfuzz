package byzzfuzz

import (
	"byzzfuzz/byzzfuzz/spec"
	"time"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
)

type MessageDrop struct {
	round int
	from  int
	to    int
}

type MessageCorruption struct {
	round      int
	to         *[]int
	corruption CorruptionType
}

type CorruptionType int

const (
	ChangeProposalToNil CorruptionType = iota
	ChangeVoteToNil
	ChangeVoteRound
)

func ByzzFuzzInst(sp *common.SystemParams, drops []MessageDrop, corruptions []MessageCorruption) *testlib.TestCase {
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

	for i := range drops {
		drop := drops[i]
		filters.AddFilter(
			testlib.If(
				testlib.IsMessageSend().
					And(isMessageFromTotalRound(drop.round)).
					And(isMessageFrom(drop.from)).
					And(isMessageTo(drop.to)),
			).Then(dropMessageLoudly()),
		)
	}

	for i := range corruptions {
		corruption := corruptions[i]
		action := actionForCorruption(corruption.corruption)

		filters.AddFilter(
			testlib.If(testlib.IsMessageSend().
				And(isMessageFromTotalRound(corruption.round)).
				And(common.IsMessageFromPart("faulty")).
				And(IsMessageToOneOf(*corruption.to)),
			).Then(action),
		)

	}

	testcase := testlib.NewTestCase("ByzzFuzzInst", 2*time.Minute, sm, filters)
	testcase.SetupFunc(common.Setup(sp))

	return testcase
}
