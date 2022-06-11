package main

import (
	"byzzfuzz/byzzfuzz"
	"byzzfuzz/byzzfuzz/spec"
	"byzzfuzz/docker"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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

	"database/sql"

	_ "modernc.org/sqlite"
)

var serverBindIp = flag.String("bind-ip", "192.167.0.1", "IP address to bind the testing server on. Should match controller-master-addr in node configuration.")
var logLevel = flag.String("log-level", "info", "Log level, one of panic|fatal|error|warn|warning|info|debug|trace")

const (
	// Main parameters for ByzzFuzz algorithm
	defaultMaxDrops       = 5
	defaultMaxCorruptions = 5
	defaultMaxSteps       = 10
)

var fuzzCmd = flag.NewFlagSet("fuzz", flag.ExitOnError)
var drops = fuzzCmd.Int("drops", defaultMaxDrops, "Bound on the number of network link faults")
var corruptions = fuzzCmd.Int("corruptions", defaultMaxCorruptions, "Bound on the number of message corruptions")
var steps = fuzzCmd.Int("steps", defaultMaxSteps, "Bound on the number of protocol consensus steps")
var timeout = fuzzCmd.Duration("timeout", 1*time.Minute, "Timeout per test instance")
var testDb = fuzzCmd.String("db", "test_results.sqlite3", "Path to test results output file")

var unittestCmd = flag.NewFlagSet("unittest", flag.ExitOnError)
var useByzzfuzz = unittestCmd.Bool("use-byzzfuzz", true, "Run unit test based on ByzzFuzz instance")

var verifyCmd = flag.NewFlagSet("verify", flag.ExitOnError)

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
	case "verify":
		verify(os.Args[commandIndex+1:])
	default:
		fmt.Println("expected 'unittest' or 'fuzz' subcommands")
		os.Exit(1)
	}
}

func unittest(args []string) {
	unittestCmd.Parse(args)
	if *useByzzfuzz {
		testcase, specCh := byzzfuzz.ByzzFuzzExpectNewRound(sysParams)
		runSingleTestCase(sysParams, testcase)
		agreementOk := !(testcase.StateMachine.CurState().Label == byzzfuzz.DiffCommitsLabel)
		if agreementOk {
			log.Println("Agreement OK")
		} else {
			log.Println("Agreement FAIL")
		}
		livenessOk := testcase.StateMachine.InSuccessState()
		if livenessOk {
			log.Println("Liveness OK")
		} else {
			log.Println("Liveness FAIL")
		}
		if spec.Check(specCh) {
			log.Println("Spec OK")
		} else {
			log.Println("Spec FAIL")
		}
	} else {
		runSingleTestCase(sysParams, byzzfuzz.ExpectNewRound(sysParams))
	}
}

type testResult struct {
	agreement bool
	spec      bool
	liveness  bool
}

func fuzz(args []string) {
	fuzzCmd.Parse(args)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	db := openTestDb()
	_ = db

	for {
		instance := byzzfuzz.ByzzFuzzRandom(sysParams, r, *drops, *corruptions, *steps, *timeout)
		log.Printf("Running test instance: %s", instance.Json())
		testcase, specCh := instance.TestCase()
		if runSingleTestCase(sysParams, testcase) {
			break
		}
		agreementOk := !(testcase.StateMachine.CurState().Label == byzzfuzz.DiffCommitsLabel)
		if agreementOk {
			log.Println("Agreement OK")
		} else {
			log.Println("Agreement FAIL")
		}
		livenessOk := testcase.StateMachine.InSuccessState()
		if livenessOk {
			log.Println("Liveness OK")
		} else {
			log.Println("Liveness FAIL")
		}
		specOk := spec.Check(specCh)
		if specOk {
			log.Println("Spec OK")
		} else {
			log.Println("SPEC FAIL")
		}
		addTestResult(db, instance, testResult{agreement: agreementOk, spec: specOk, liveness: livenessOk})
	}
}

func verify(args []string) {
	verifyCmd.Parse(args)
	inst := byzzfuzz.Bug003Reprod()

	testcase, specCh := inst.TestCase()
	runSingleTestCase(sysParams, testcase)
	agreementOk := !(testcase.StateMachine.CurState().Label == byzzfuzz.DiffCommitsLabel)
	if agreementOk {
		log.Println("Agreement OK")
	} else {
		log.Println("Agreement FAIL")
	}
	livenessOk := testcase.StateMachine.InSuccessState()
	if livenessOk {
		log.Println("Liveness OK")
	} else {
		log.Println("Liveness FAIL")
	}
	if spec.Check(specCh) {
		log.Println("Spec OK")
	} else {
		log.Println("Spec FAIL")
	}
}

func openTestDb() *sql.DB {
	db, err := sql.Open("sqlite", *testDb)
	if err != nil {
		log.Fatalf("failed to open test database: %s", err.Error())
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS TestResults(
			config JSON,
			agreement BOOL,
			spec BOOL,
			liveness BOOL);
		CREATE TABLE IF NOT EXISTS SpecLogs(
			test_id INT,
			log TEXT);
	`)
	if err != nil {
		log.Fatalf("failed to create test database: %s", err.Error())
	}

	return db
}

func addTestResult(db *sql.DB, instance byzzfuzz.ByzzFuzzInstanceConfig, result testResult) {
	res, err := db.Exec("INSERT INTO TestResults VALUES (?, ?, ?, ?)", instance.Json(), result.agreement, result.spec, result.liveness)
	if err != nil {
		log.Fatalf("failed to write to DB: %s", err.Error())
	}
	rowid, err := res.LastInsertId()
	if err != nil {
		log.Fatalf("no rowid returned")
	}

	// Add spec logs
	specLogsB, err := os.ReadFile("spec.log")
	if err != nil {
		log.Fatalf("failed to read spec logs")
	}
	specLogs := string(specLogsB)
	_, err = db.Exec("INSERT INTO SpecLogs VALUES (?, ?)", rowid, specLogs)
	if err != nil {
		log.Fatalf("failed to write spec logs to DB: %s", err.Error())
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

	stopRequests := make(chan bool)
	go func() {
		for i := 0; ; i++ {
			time.Sleep(5 * time.Second)
			select {
			case <-stopRequests:
				return
			default:
			}
			res, err := http.Get(fmt.Sprintf("http://localhost:26657/broadcast_tx_commit?tx=\"name=satoshi%d\"", time.Now().UnixNano()))
			if err != nil {
				log.Printf("Error sending request: %s", err.Error())
				continue
			}

			log.Printf("Request status: %s", res.Status)
			/*
				bodyB, err := io.ReadAll(res.Body)
				if err != nil {
					log.Fatalf("Failed to read response: %s", err.Error())
				}
				log.Printf("Response: %s", string(bodyB))
				defer res.Body.Close()
			*/

		}
	}()

	doneCh := server.Done()
	terminate = false
	go func() {
		select {
		case <-termCh:
			terminate = true
			server.Stop()
			stopRequests <- true
		case <-doneCh:
			server.Stop()
			stopRequests <- true
		}

		close(stopRequests)
	}()

	// Returns once the server has been stopped
	server.Start()

	log.Printf("Stopping nodes...")
	dockerCompose.Process.Signal(syscall.SIGTERM)
	dockerCompose.Wait()

	return terminate
}
