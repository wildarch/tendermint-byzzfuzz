package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/ImperiumProject/imperium/config"
	"github.com/ImperiumProject/imperium/testlib"
	"github.com/ImperiumProject/imperium/types"
	"github.com/ImperiumProject/tendermint-test/common"
	"github.com/ImperiumProject/tendermint-test/util"
)

func main() {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	doneCh := make(chan bool, 1)

	sysParams := common.NewSystemParams(4)

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
			HigherRound(sysParams, doneCh),
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

func HigherRound(sp *common.SystemParams, doneChan chan bool) *testlib.TestCase {
	sm := testlib.NewStateMachine()

	cascade := testlib.NewHandlerCascade()
	cascade.AddHandler(
		testlib.If(common.IsMessageFromRound(0).
			And(common.IsMessageToPart("h")),
		).Then(testlib.DropMessage()),
	)
	cascade.AddHandler(
		testlib.If(common.IsMessageFromRound(1).
			And(common.IsMessageToPart("h")),
		).Then(
			testlib.Count("higherRoundMessage").Incr(),
			testlib.DeliverMessage(),
		),
	)

	init := sm.Builder()
	expectRoundChange := init.On(
		testlib.Count("higherRoundMessage").Gt(sp.F),
		"expectRoundChange",
	)
	expectRoundChange.On(
		ReplicaNewRound("h", 1),
		testlib.SuccessStateLabel,
	)

	testcase := testlib.NewTestCase("HigherRound", 30*time.Second, sm, cascade)
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

func ReplicaNewRound(partS string, round int) testlib.Condition {
	return func(e *types.Event, c *testlib.Context) bool {
		p, exists := c.Vars.Get("partition")
		if !exists {
			return false
		}
		partition, ok := p.(*util.Partition)
		if !ok {
			return false
		}
		part, ok := partition.GetPart(partS)
		if !ok {
			return false
		}

		return part.Contains(e.Replica) && common.IsEventNewRound(round)(e, c)
	}
}
