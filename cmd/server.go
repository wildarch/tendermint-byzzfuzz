package main

import (
	"byzzfuzz/byzzfuzz"
	"byzzfuzz/docker"
	"flag"
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
	"github.com/netrixframework/tendermint-testing/testcases/rskip"
	"github.com/netrixframework/tendermint-testing/util"
)

var serverBindIp = flag.String("bind-ip", "192.167.0.1", "IP address to bind the testing server on. Should match controller-master-addr in node configuration.")

func main() {
	sysParams := common.NewSystemParams(4)
	//testcase := rskip.ExpectNewRound(sysParams)
	//testcase := rskip.RoundSkip(sysParams, 1, 2)
	testcase := byzzfuzz.ByzzFuzzExpectNewRound(sysParams)
	runSingleTestCase(sysParams, testcase)
}

func runSingleTestCase(sysParams *common.SystemParams, testcase *testlib.TestCase) {
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
	go func() {
		select {
		case <-termCh:
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
}

func expectNewRound(sp *common.SystemParams) testlib.TestCase {
	return *rskip.ExpectNewRound(sp)
}
