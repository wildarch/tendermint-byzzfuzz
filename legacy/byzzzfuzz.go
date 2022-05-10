package legacy

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ImperiumProject/imperium/config"
	"github.com/ImperiumProject/imperium/testlib"
	"github.com/ImperiumProject/imperium/types"
	"github.com/ImperiumProject/tendermint-test/common"
	"github.com/ImperiumProject/tendermint-test/util"
)

func LegacyMain() {

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	doneCh := make(chan bool, 1)

	sysParams := common.NewSystemParams(4)
	random := rand.New(rand.NewSource(42))
	corruptions := 5
	networkFaults := 10
	rounds := 5

	server, err := testlib.NewTestingServer(
		&config.Config{
			APIServerAddr: "192.167.0.1:7074",
			NumReplicas:   sysParams.N,
			LogConfig: config.LogConfig{
				Format: "json",
				Path:   "/tmp/tendermint/log/checker.log",
			},
		},
		&util.TMessageParser{},
		[]*testlib.TestCase{
			ByzzFuzz(sysParams, random, corruptions, networkFaults, rounds, doneCh),
		},
	)

	if err != nil {
		fmt.Printf("Failed to start server: %s\n", err.Error())
		os.Exit(1)
	}

	// Stdout to file
	dockerCompose := exec.Command("make", "localnet-start")
	dockerCompose.Dir = "third_party/tendermint-pct-instrumentation"
	stdoutFile, err := os.Create("nodes.stdout.log")
	if err != nil {
		log.Fatalf("Cannot create stdout file: %v", err)
	}
	defer stdoutFile.Close()
	dockerCompose.Stdout = stdoutFile
	dockerCompose.Stderr = stdoutFile

	err = dockerCompose.Start()
	if err != nil {
		log.Fatalf("Failed to start nodes: %v", err)
	}

	go func() {
		select {
		case <-termCh:
		case <-doneCh:
			server.Stop()
		}
	}()

	server.Start()
	// Returns once the server has been stopped

	dockerCompose.Process.Signal(syscall.SIGTERM)
	dockerCompose.Wait()

}

const maxHeight = 3

func ByzzFuzz(sp *common.SystemParams, random *rand.Rand, corruptions int, networkFaults int, rounds int, doneChan chan bool) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	maxHeightReached := init.On(common.HeightReached(maxHeight), "maxHeightReached")
	maxHeightReached.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)
	maxHeightReached.On(
		common.IsCommit(),
		testlib.SuccessStateLabel,
	)

	cascade := testlib.NewHandlerCascade()
	cascade.AddHandler(trackGlobalRound)

	for i := 0; i < networkFaults; i++ {
		round := random.Intn(rounds)
		from := random.Intn(sp.N)
		to := random.Intn(sp.N)
		// Drop messages matching round, from, to
		log.Printf("Will drop messages (from=%d, to=%d, round=%d)", from, to, round)
		cascade.AddHandler(
			testlib.If(
				testlib.IsMessageSend().
					And(isMessageFromGlobalRound(round)).
					And(isMessageFrom(from)).
					And(isMessageTo(to)),
			).Then(dropMessageLoudly()),
		)
	}

	testcase := testlib.NewTestCase("ByzzFuzz", 2*time.Minute, sm, cascade)
	testcase.SetupFunc(common.Setup(sp))

	// Send done signal once test is in success state
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			if sm.InSuccessState() {
				doneChan <- true
				return
			}
		}
	}()
	return testcase

}

func dropMessageLoudly() testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		log.Printf("Dropping message!")
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

func isMessageFromGlobalRound(round int) testlib.Condition {
	return func(e *types.Event, c *testlib.Context) bool {
		m, ok := util.GetMessageFromEvent(e, c)
		if !ok {
			return false
		}
		roundsTillLastCommit, ok := c.Vars.GetInt(roundKey)
		if !ok {
			roundsTillLastCommit = 0
		}

		return (roundsTillLastCommit + m.Round()) == round
	}
}

const heightKey = "BF_height"
const roundKey = "BF_round"

func trackGlobalRound(e *types.Event, c *testlib.Context) (messages []*types.Message, handled bool) {
	eType, ok := e.Type.(*types.GenericEventType)
	if !ok {
		return
	}
	if eType.T != "Committing block" {
		return
	}
	// Round
	roundS, ok := eType.Params["round"]
	if !ok {
		panic("Cannot read round")
	}
	round, err := strconv.Atoi(roundS)
	if err != nil {
		panic(err)
	}
	// Height
	heightS, ok := eType.Params["height"]
	if !ok {
		panic("Cannot read height")
	}
	height, err := strconv.Atoi(heightS)
	if err != nil {
		panic(err)
	}

	prevHeight, ok := c.Vars.GetInt(heightKey)
	if !ok {
		prevHeight = -1
	}
	if prevHeight == height {
		// Already updated round
		return
	}
	c.Vars.Set(heightKey, height)
	prevRound, ok := c.Vars.GetInt(roundKey)
	if !ok {
		prevRound = 0
	}
	newRound := prevRound + round + 1
	log.Printf("New global round: %d (prevRound = %d, round = %d)", newRound, prevRound, round)
	c.Vars.Set(roundKey, newRound)
	return
}

func CommitAfterRoundSkip(sp *common.SystemParams) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()

	roundOne := init.On(
		common.RoundReached(1),
		"Round1",
	)
	roundOne.On(
		common.IsCommitForProposal("zeroProposal"),
		testlib.SuccessStateLabel,
	)
	roundOne.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)

	cascade := testlib.NewHandlerCascade()
	cascade.AddHandler(common.TrackRoundAll)
	cascade.AddHandler(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsVoteFromFaulty()),
		).Then(
			common.ChangeVoteToNil(),
		),
	)
	cascade.AddHandler(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsVoteFromPart("h")),
		).Then(
			testlib.Set("delayedVotes").Store(),
			testlib.DropMessage(),
		),
	)
	cascade.AddHandler(
		testlib.If(
			testlib.IsMessageSend().Not().
				And(sm.InState("Round1")),
		).Then(
			testlib.Set("delayedVotes").DeliverAll(),
			testlib.DeliverMessage(),
		),
	)
	cascade.AddHandler(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsMessageFromRound(0)).
				And(common.IsMessageType(util.Proposal)),
		).Then(
			common.RecordProposal("zeroProposal"),
			testlib.DeliverMessage(),
		),
	)

	testcase := testlib.NewTestCase(
		"CommitAfterRoundSkip",
		2*time.Minute,
		sm,
		cascade,
	)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}

func RoundSkip(sysParams *common.SystemParams, height, round int) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	roundReached := sm.Builder().
		On(common.HeightReached(height), "SkipRounds").
		On(common.RoundReached(round), "roundReached")

	roundReached.MarkSuccess()
	roundReached.On(
		common.DiffCommits(),
		testlib.FailStateLabel,
	)

	cascade := testlib.NewHandlerCascade()
	cascade.AddHandler(common.TrackRoundAll)
	cascade.AddHandler(
		testlib.If(
			common.IsFromHeight(height).Not(),
		).Then(
			testlib.DeliverMessage(),
		),
	)
	cascade.AddHandler(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsFromHeight(height)).
				And(common.IsVoteFromFaulty()),
		).Then(
			common.ChangeVoteToNil(),
		),
	)
	cascade.AddHandler(
		testlib.If(
			sm.InState("roundReached"),
		).Then(
			testlib.Set("DelayedPrevotes").DeliverAll(),
		),
	)
	cascade.AddHandler(
		testlib.If(
			testlib.IsMessageSend().
				And(common.IsFromHeight(height)).
				And(common.IsMessageFromPart("h")).
				And(common.IsMessageType(util.Prevote)),
		).Then(
			testlib.Set("DelayedPrevotes").Store(),
			testlib.DropMessage(),
		),
	)

	testCase := testlib.NewTestCase(
		"RoundSkipWithPrevotes",
		30*time.Second,
		sm,
		cascade,
	)
	testCase.SetupFunc(common.Setup(sysParams))
	return testCase
}
