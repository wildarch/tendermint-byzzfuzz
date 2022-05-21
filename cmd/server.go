package main

import (
	"byzzfuzz/netrix"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/testcases/byzantine"
	"github.com/netrixframework/tendermint-testing/testcases/invariant"
	"github.com/netrixframework/tendermint-testing/testcases/lockedvalue"
	"github.com/netrixframework/tendermint-testing/testcases/mainpath"
	"github.com/netrixframework/tendermint-testing/testcases/rskip"
	"github.com/netrixframework/tendermint-testing/util"
)

func main() {
	//legacy.LegacyMain()
	netrix.Main()
	//runUnitTests()
}

func runUnitTests() {
	sysParams := common.NewSystemParams(4)

	testcases := []*testlib.TestCase{
		rskip.RoundSkip(sysParams, 1, 2),
		rskip.BlockVotes(sysParams),
		rskip.CommitAfterRoundSkip(sysParams),
		lockedvalue.DifferentDecisions(sysParams),
		lockedvalue.ExpectUnlock(sysParams),
		lockedvalue.LockedCommit(sysParams),
		mainpath.NilPrevotes(sysParams),
		mainpath.ProposalNilPrevote(sysParams),
		mainpath.ProposePrevote(sysParams),
		mainpath.QuorumPrevotes(sysParams),
		invariant.NotNilDecide(sysParams),
		byzantine.HigherRound(sysParams),
		byzantine.CrashReplica(sysParams),
		invariant.PrecommitsInvariant(sysParams),

		byzantine.LaggingReplica(sysParams, 10, 10*time.Minute),
		rskip.ExpectNewRound(sysParams),
		lockedvalue.ExpectNoUnlock(sysParams),
		lockedvalue.Relocked(sysParams),
		invariant.QuorumPrecommits(sysParams),
		byzantine.GarbledMessage(sysParams),
		byzantine.ForeverLaggingReplica(sysParams),
	}

	for i := range testcases {
		if runTestCase(sysParams, testcases[i]) {
			break
		}
	}
}

func runTestCase(sp *common.SystemParams, testcase *testlib.TestCase) bool {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	server, err := testlib.NewTestingServer(
		&config.Config{
			APIServerAddr: "192.167.0.1:7074",
			NumReplicas:   sp.N,
			LogConfig: config.LogConfig{
				Format: "json",
				Path:   "/tmp/tendermint/log/checker.log",
			},
		},
		&util.TMessageParser{},
		[]*testlib.TestCase{testcase},
	)

	if err != nil {
		fmt.Printf("Failed to start server: %s\n", err.Error())
		os.Exit(1)
	}
	prepDockerCompose()
	// Stdout to file
	dockerCompose := exec.Command("make", "localnet-start")
	dockerCompose.Dir = "third_party/tendermint-pct-instrumentation"
	stdoutFile, err := os.Create(testcase.Name + "_nodes.stdout.log")
	if err != nil {
		log.Fatalf("Cannot create stdout file: %v", err)
	}
	defer stdoutFile.Close()
	dockerCompose.Stdout = stdoutFile
	dockerCompose.Stderr = stdoutFile

	go func() {
		log.Printf("Start nodes now!")
		time.Sleep(5 * time.Second)
		log.Printf("Starting nodes")
		err = dockerCompose.Start()
		if err != nil {
			log.Fatalf("Failed to start nodes: %v", err)
		}
	}()

	terminate := false
	doneCh := server.Done()
	go func() {
		select {
		case <-termCh:
			terminate = true
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

	return terminate
}

func prepDockerCompose() {
	localNetStop := exec.Command("make", "localnet-stop")
	localNetStop.Dir = "third_party/tendermint-pct-instrumentation"
	err := localNetStop.Run()
	if err != nil {
		log.Fatalf("Failed to stop previous local net: %v", err)
	}

	dockerComposeUpNoStart := exec.Command("docker-compose", "up", "--no-start")
	dockerComposeUpNoStart.Dir = "third_party/tendermint-pct-instrumentation"
	err = dockerComposeUpNoStart.Run()
	if err != nil {
		log.Fatalf("Failed to prepare network: %v", err)
	}
}
