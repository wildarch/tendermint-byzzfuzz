package main

import (
	"byzzfuzz/byzzfuzz"
	"byzzfuzz/docker"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/testcases/rskip"
	"github.com/netrixframework/tendermint-testing/util"
)

var serverBindIp = flag.String("bind-ip", "192.167.0.1", "IP address to bind the testing server on. Should match controller-master-addr in node configuration.")

var unittestCmd = flag.NewFlagSet("unittest", flag.ExitOnError)
var fuzzCmd = flag.NewFlagSet("fuzz", flag.ExitOnError)

const (
	// Main parameters for ByzzFuzz algorithm
	defaultMaxDrops       = 10
	defaultMaxCorruptions = 5
	defaultMaxRounds      = 5
)

var maxDrops = fuzzCmd.Int("max-drops", defaultMaxDrops, "Bound on the number of network link faults")
var maxCorruptions = fuzzCmd.Int("max-corruptions", defaultMaxCorruptions, "Bound on the number of message corruptions")
var maxRounds = fuzzCmd.Int("max-rounds", defaultMaxRounds, "Bound on the number of protocol rounds")
var timeout = fuzzCmd.Duration("timeout", 2*time.Minute, "Timeout per test instance")

var sysParams = common.NewSystemParams(4)

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf("Usage: %s unittest|fuzz\n", os.Args[0])
		os.Exit(1)
	}
	switch os.Args[1] {
	case "unittest":
		unittest()
	case "fuzz":
		fuzz()
	default:
		fmt.Println("expected 'unittest' or 'fuzz' subcommands")
		os.Exit(1)
	}
	testcase := byzzfuzz.ByzzFuzzExpectNewRound(sysParams)
	runSingleTestCase(sysParams, testcase)
}

func unittest() {
	unittestCmd.Parse(os.Args[2:])
	runSingleTestCase(sysParams, rskip.ExpectNewRound(sysParams))
}

func fuzz() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		instance := byzzfuzz.ByzzFuzzRandom(sysParams, r, *maxDrops, *maxCorruptions, *maxRounds, *timeout)
		log.Printf("Running test instance: %s", instance.Json())
		if runSingleTestCase(sysParams, instance.TestCase()) {
			break
		}
	}
}

func runSingleTestCase(sysParams *common.SystemParams, testcase *testlib.TestCase) (terminate bool) {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	server, err := testlib.NewTestingServer(
		&config.Config{
			APIServerAddr: fmt.Sprintf("%s:7074", *serverBindIp),
			NumReplicas:   sysParams.N,
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

	docker.PrepDockerCompose()

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

	doneCh := server.Done()
	terminate = false
	go func() {
		select {
		case <-termCh:
			terminate = true
			server.Stop()
		case <-doneCh:
			server.Stop()
		}
	}()

	// Returns once the server has been stopped
	server.Start()

	log.Printf("Stopping nodes...")
	dockerCompose.Process.Signal(syscall.SIGTERM)
	dockerCompose.Wait()

	return terminate
}

func expectNewRound(sp *common.SystemParams) testlib.TestCase {
	return *rskip.ExpectNewRound(sp)
}
