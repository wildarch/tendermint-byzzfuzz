package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

func main() {

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	sysParams := common.NewSystemParams(4)
	random := rand.New(rand.NewSource(42))
	corruptions := 5
	networkFaults := 10
	rounds := 10

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
			ByzzFuzz(sysParams, random, corruptions, networkFaults, rounds),
		},
	)

	if err != nil {
		fmt.Printf("Failed to start server: %s\n", err.Error())
		os.Exit(1)
	}

	go func() {
		<-termCh
		server.Stop()
	}()

	server.Start()

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

func isMessageToOneOf(replicaIdxs []int) testlib.Condition {
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

func corruptMessage(seed rand.Source) testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		if !e.IsMessageSend() {
			return []*types.Message{}
		}
		message, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(message)
		switch tMsg.Type {
		// Main consensus steps
		case util.Proposal:
			return []*types.Message{corruptProposal(seed, c, tMsg)}
		case util.Prevote:
		case util.Precommit:
			return []*types.Message{corruptVote(seed, c, tMsg)}
		// Note sure we care about these
		case util.NewRoundStep:
			panic("NewRoundStep")
		case util.NewValidBlock:
			panic("NewValidBlock")
		case util.ProposalPol:
			panic("ProposalPol")
		case util.BlockPart:
			panic("BlockPart")
		case util.Vote:
			panic("Vote")
		case util.HasVote:
			panic("HasVote")
		case util.VoteSetMaj23:
			panic("VoteSetMaj23")
		case util.VoteSetBits:
			panic("VoteSetBits")
		case util.None:
			panic("None")
		}
		if !ok {
			return []*types.Message{}
		}

		return nil
	}
}

func corruptProposal(seed rand.Source, c *testlib.Context, tMsg *util.TMessage) *types.Message {
	// Things to corrupt:
	// * Height
	// * Round
	// * POLRound
	// * BlockID
	// * Timestamp
	// * Signature
	panic("corruptProposal")
}

func corruptVote(seed rand.Source, c *testlib.Context, tMsg *util.TMessage) *types.Message {
	// Things to corrupt:
	// * Height
	// * Round
	// * BlockID
	// * Timestamp
	// * Signature
	panic("corruptPrevote")
}

func ByzzFuzz(sp *common.SystemParams, random *rand.Rand, corruptions int, networkFaults int, rounds int) *testlib.TestCase {
	sm := testlib.NewStateMachine()
	init := sm.Builder()
	init.On(
		common.RoundReached(rounds),
		testlib.SuccessStateLabel,
	)

	filters := testlib.NewFilterSet()
	// Sample network faults.
	for i := 0; i < networkFaults; i++ {
		round := random.Intn(rounds)
		from := random.Intn(sp.N)
		to := random.Intn(sp.N)
		// Drop messages matching round, from, to
		filters.AddFilter(
			testlib.If(
				testlib.IsMessageSend().
					And(common.IsMessageFromRound(round)).
					And(isMessageFrom(from)).
					And(isMessageTo(to)),
			).Then(testlib.DropMessage()),
		)
	}

	// Sample faulty replicate
	faultyReplica := random.Intn(sp.N)

	// Sample corruptions.
	for i := 0; i < corruptions; i++ {
		round := random.Intn(rounds)
		// Random subset of replica indices
		// TODO: Check if this is correct
		procs := random.Perm(sp.N)[0:random.Intn(sp.N)]
		corSeed := rand.NewSource(random.Int63())

		filters.AddFilter(
			testlib.If(testlib.IsMessageSend().
				And(common.IsMessageFromRound(round)).
				And(isMessageFrom(faultyReplica)).
				And(isMessageToOneOf(procs)),
			).Then(
				corruptMessage(corSeed),
			),
		)
	}

	testcase := testlib.NewTestCase("ByzzFuzz", 2*time.Minute, sm, filters)
	testcase.SetupFunc(common.Setup(sp))
	return testcase
}
