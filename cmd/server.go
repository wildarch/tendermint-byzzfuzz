package main

import (
	"byzzfuzz/byzzfuzz"
	"byzzfuzz/byzzfuzz/spec"
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
var maxDrops = fuzzCmd.Int("max-drops", defaultMaxDrops, "Bound on the number of network link faults")
var maxCorruptions = fuzzCmd.Int("max-corruptions", defaultMaxCorruptions, "Bound on the number of message corruptions")
var maxSteps = fuzzCmd.Int("max-steps", defaultMaxSteps, "Bound on the number of protocol consensus steps")
var timeout = fuzzCmd.Duration("timeout", 2*time.Minute, "Timeout per test instance")
var testDb = fuzzCmd.String("db", "test_results.sqlite3", "Path to test results output file")

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
		testcase, specCh := byzzfuzz.ByzzFuzzExpectNewRound(sysParams)
		runSingleTestCase(sysParams, testcase)
		spec.Check(specCh)
	} else {
		runSingleTestCase(sysParams, byzzfuzz.ExpectNewRound(sysParams))
	}
}

func fuzz(args []string) {
	fuzzCmd.Parse(args)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	db := openTestDb()
	_ = db

	for {
		instance := byzzfuzz.ByzzFuzzRandom(sysParams, r, *maxDrops, *maxCorruptions, *maxSteps, *timeout)
		log.Printf("Running test instance: %s", instance.Json())
		testcase, specCh := instance.TestCase()
		if runSingleTestCase(sysParams, testcase) {
			break
		}
		spec.Check(specCh)
		success := testcase.StateMachine.InSuccessState()
		if success {
			log.Println("Testcase succesful!")
		} else {
			log.Println("Testcase failed")
		}
		addTestResult(db, instance, success)
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
			pass INT,
			fail INT);
	`)
	if err != nil {
		log.Fatalf("failed to create test data: %s", err.Error())
	}

	return db
}

func addTestResult(db *sql.DB, instance byzzfuzz.ByzzFuzzInstanceConfig, success bool) {
	row := db.QueryRow("SELECT rowid, pass, fail FROM TestResults WHERE config = ?", instance.Json())
	rowid := 0
	pass := 0
	fail := 0
	err := row.Scan(&rowid, &pass, &fail)
	if err == sql.ErrNoRows {
		if success {
			pass++
		} else {
			fail++
		}
		_, err = db.Exec("INSERT INTO TestResults VALUES (?, ?, ?)", instance.Json(), pass, fail)
		if err != nil {
			log.Fatalf("failed to write to DB")
		}
	} else {
		db.Exec("UPDATE TestResults SET pass=? fail=? WHERE rowid=?", pass, fail, rowid)
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
