package netrix

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"
)

func Main() {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	doneCh := make(chan bool, 1)

	sysParams := common.NewSystemParams(4)
	random := rand.New(rand.NewSource(43))
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

	go func() {
		time.Sleep(5 * time.Second)
		log.Printf("Starting nodes")
		err = dockerCompose.Start()
		if err != nil {
			log.Fatalf("Failed to start nodes: %v", err)
		}
	}()

	go func() {
		select {
		case <-termCh:
			server.Stop()
		case <-doneCh:
			server.Stop()
		}
	}()

	server.Start()
	// Returns once the server has been stopped

	log.Printf("Stopping nodes...")
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
	// TODO: Check if we expect consensus to be possible based on number of network faults
	maxHeightReached.On(
		common.IsCommit(),
		testlib.SuccessStateLabel,
	)

	cascade := testlib.NewFilterSet()
	cascade.AddFilter(trackGlobalRound)

	// Samples drops
	for i := 0; i < networkFaults; i++ {
		round := random.Intn(rounds)
		from := random.Intn(sp.N)
		to := random.Intn(sp.N)
		// Drop messages matching round, from, to
		log.Printf("Will drop messages (from=%d, to=%d, round=%d)", from, to, round)
		cascade.AddFilter(
			testlib.If(
				testlib.IsMessageSend().
					And(isMessageFromGlobalRound(round)).
					And(isMessageFrom(from)).
					And(isMessageTo(to)),
			).Then(dropMessageLoudly()),
		)
	}

	// Sample corruptions.
	for i := 0; i < corruptions; i++ {
		round := random.Intn(rounds)
		// Random subset of replica indices
		// TODO: Check if this is correct
		procs := random.Perm(sp.N)[0:random.Intn(sp.N)]
		corRandom := rand.New(rand.NewSource(random.Int63()))

		log.Printf("Will corrupt messages (from=faulty, to=%v, round=%d, seed=%v)", procs, round, corRandom)
		cascade.AddFilter(
			testlib.If(testlib.IsMessageSend().
				And(isMessageFromGlobalRound(round)).
				And(common.IsMessageFromPart("faulty")).
				And(IsMessageToOneOf(procs)),
			).Then(
				corruptMessage(corRandom),
			),
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

func corruptMessage(random *rand.Rand) testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		m, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(m)
		if !ok {
			return []*types.Message{m}
		}

		// Debugging
		from := -1
		to := -1
		for i, r := range c.Replicas.Iter() {
			if r.ID == tMsg.From {
				from = i
			}
			if r.ID == tMsg.To {
				to = i
			}
		}
		roundsTillLastCommit, ok := c.Vars.GetInt(roundKey)
		if !ok {
			roundsTillLastCommit = 0
		}
		globalRound := roundsTillLastCommit + tMsg.Round()
		log.Printf("Corrupting message (from=%d, to=%d, round=%d)", from, to, globalRound)

		switch tMsg.Type {
		case util.Proposal:
			// TODO:
			// - ChangeProposalLockValue (TODO?)
			actions := [1]testlib.Action{
				changeProposalToNil,
			}
			return actions[random.Intn(len(actions))](e, c)
		case util.Precommit:
			fallthrough
		case util.Prevote:
			// TODO:
			// - ChangeVoteToProposalMessage (see relocked)
			// - ChangeVoteTime
			actions := [2]testlib.Action{
				common.ChangeVoteToNil(),
				changeVoteRound(),
			}
			return actions[random.Intn(len(actions))](e, c)
		default:
			return []*types.Message{m}
		}
	}
}

func changeVoteRound() testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		m, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(m)
		if !ok {
			return []*types.Message{m}
		}
		if tMsg.Type != util.Precommit && tMsg.Type != util.Prevote {
			return []*types.Message{}
		}
		valAddr, ok := util.GetVoteValidator(tMsg)
		if !ok {
			return []*types.Message{}
		}
		var replica *types.Replica = nil
		for _, r := range c.Replicas.Iter() {
			addr, err := util.GetReplicaAddress(r)
			if err != nil {
				continue
			}
			if bytes.Equal(addr, valAddr) {
				replica = r
				break
			}
		}
		if replica == nil {
			return []*types.Message{}
		}
		newVote, err := util.ChangeVoteRound(replica, tMsg, int32(tMsg.Round()+2))
		if err != nil {
			return []*types.Message{}
		}
		msgB, err := newVote.Marshal()
		if err != nil {
			return []*types.Message{}
		}
		return []*types.Message{c.NewMessage(m, msgB)}
	}
}

func changeProposalToNil(e *types.Event, c *testlib.Context) []*types.Message {
	message, _ := c.GetMessage(e)
	tMsg, ok := util.GetParsedMessage(message)
	if !ok {
		return []*types.Message{}
	}
	replica, _ := c.Replicas.Get(tMsg.From)
	newProp, err := util.ChangeProposalBlockIDToNil(replica, tMsg)
	if err != nil {
		//c.Logger().With(log.LogParams{"error": err}).Error("Failed to change proposal")
		return []*types.Message{message}
	}
	newMsgB, err := newProp.Marshal()
	if err != nil {
		//c.Logger().With(log.LogParams{"error": err}).Error("Failed to marshal changed proposal")
		return []*types.Message{message}
	}
	return []*types.Message{c.NewMessage(message, newMsgB)}
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
			roundsTillLastCommit, ok := c.Vars.GetInt(roundKey)
			if !ok {
				roundsTillLastCommit = 0
			}
			globalRound := roundsTillLastCommit + message.Round()
			log.Printf("Dropping message (from=%d, to=%d, round=%d)", from, to, globalRound)
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

func isMessageFromGlobalRound(round int) testlib.Condition {
	return func(e *types.Event, c *testlib.Context) bool {
		m, ok := util.GetMessageFromEvent(e, c)
		if !ok {
			return false
		}
		if m.Round() < 0 {
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
