package main

import (
	"byzzfuzz/byzzfuzz"
	"byzzfuzz/docker"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/tendermint-testing/common"
	"github.com/netrixframework/tendermint-testing/util"

	_ "modernc.org/sqlite"
)

var serverBindIp = flag.String("bind-ip", "192.167.0.1", "IP address to bind the testing server on. Should match controller-master-addr in node configuration.")
var logLevel = flag.String("log-level", "info", "Log level, one of panic|fatal|error|warn|warning|info|debug|trace")

var runInstanceCmd = flag.NewFlagSet("run-instance", flag.ExitOnError)
var livenessTimeout = runInstanceCmd.Duration("liveness-timeout", 1*time.Minute, "Time to wait for a new commit after the network heals, to verify liveness")

var baselineCmd = flag.NewFlagSet("baseline", flag.ExitOnError)
var dropPercent = baselineCmd.Int("drop-percent", 25, "Percentage of messages to drop (e.g. 25 for 25%)")
var corruptPercent = baselineCmd.Int("corrupt-percent", 25, "Percentage of messages to corrupt (e.g. 25 for 25%)")

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
		fmt.Printf("Usage: %s run-instance|baseline\n", os.Args[0])
		os.Exit(1)
	}
	switch os.Args[commandIndex] {
	case "run-instance":
		runInstance(os.Args[commandIndex+1:])
	case "baseline":
		baseline(os.Args[commandIndex+1:])
	default:
		fmt.Println("expected 'run-instance' or 'baseline' subcommands")
		os.Exit(1)
	}
}

func runInstance(args []string) {
	runInstanceCmd.Parse(args)

	// Read testcase from stdin
	instConf, err := byzzfuzz.InstanceFromJson(os.Stdin)
	if err != nil {
		log.Fatalf("failed to parse JSON definition for instance: %s", err.Error())
	}
	instConf.LivenessTimeout = *livenessTimeout
	testcase := instConf.TestCase()

	confB, err := json.Marshal(instConf)
	if err != nil {
		log.Fatal(err)
	}
	confB = append(confB, byte('\n'))
	_, err = os.Stderr.Write(confB)
	if err != nil {
		log.Fatal(err)
	}

	runSingleTestCase(sysParams, testcase)
}

func baseline(args []string) {
	baselineCmd.Parse(args)

	testcase := byzzfuzz.BaselineTestCase(sysParams, *dropPercent, *corruptPercent)

	runSingleTestCase(sysParams, testcase)
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
		server.Logger.Info("Starting nodes")
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

	server.Logger.Info("Stopping nodes")
	dockerCompose.Process.Signal(syscall.SIGTERM)
	dockerCompose.Wait()

	return terminate
}
