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
	"strings"
	"syscall"
	"time"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/testcases/rskip"
	"github.com/netrixframework/tendermint-testing/util"
)

var serverBindIp = flag.String("bind-ip", "192.167.0.1", "IP address to bind the testing server on. Should match controller-master-addr in node configuration.")
var logLevel = flag.String("log-level", "info", "Log level, one of panic|fatal|error|warn|warning|info|debug|trace")

const (
	// Main parameters for ByzzFuzz algorithm
	defaultMaxDrops       = 3
	defaultMaxCorruptions = 5
	defaultMaxSteps       = 10
)

var fuzzCmd = flag.NewFlagSet("fuzz", flag.ExitOnError)
var maxDrops = fuzzCmd.Int("max-drops", defaultMaxDrops, "Bound on the number of network link faults")
var maxCorruptions = fuzzCmd.Int("max-corruptions", defaultMaxCorruptions, "Bound on the number of message corruptions")
var maxSteps = fuzzCmd.Int("max-steps", defaultMaxSteps, "Bound on the number of protocol consensus steps")
var timeout = fuzzCmd.Duration("timeout", 2*time.Minute, "Timeout per test instance")

var unittestCmd = flag.NewFlagSet("unittest", flag.ExitOnError)
var useByzzfuzz = unittestCmd.Bool("use-byzzfuzz", true, "Run unit test based on ByzzFuzz instance")

var sysParams = common.NewSystemParams(4)

func main() {
	flag.Parse()
	commandIndex := 1
	for _, v := range os.Args[1:] {
		if !strings.HasPrefix(v, "-") {
			break
		}
		commandIndex++
	}
	if len(os.Args) <= commandIndex {
		fmt.Printf("Usage: %s unittest|fuzz\n", os.Args[0])
		os.Exit(1)
	}
	switch os.Args[commandIndex] {
	case "unittest":
		unittest(os.Args[commandIndex+1:])
	case "fuzz":
		fuzz(os.Args[commandIndex+1:])
	default:
		fmt.Println("expected 'unittest' or 'fuzz' subcommands")
		os.Exit(1)
	}
}

func unittest(args []string) {
	unittestCmd.Parse(args)
	if *useByzzfuzz {
		runSingleTestCase(sysParams, byzzfuzz.ByzzFuzzExpectNewRound(sysParams))
	} else {
		runSingleTestCase(sysParams, rskip.ExpectNewRound(sysParams))
	}
}

func fuzz(args []string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		instance := byzzfuzz.ByzzFuzzRandom(sysParams, r, *maxDrops, *maxCorruptions, *maxSteps, *timeout)
		log.Printf("Running test instance: %s", instance.Json())
		testcase := instance.TestCase()
		if runSingleTestCase(sysParams, testcase) {
			break
		}
		if testcase.StateMachine.InSuccessState() {
			log.Println("Testcase succesful!")
		} else {
			log.Println("Testcase failed")
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
				Level:  *logLevel,
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
