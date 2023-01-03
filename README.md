# Getting started
## Requirements
- Golang 1.18
- Docker
- [docker-compose](https://pypi.org/project/docker-compose/). Not the one built-in to recent versions of docker. I found pip to be the easiest way to install it: `pip3 install --user docker-compose`.
- SQLite

## Setup
Some setup is required to prepare the modified Tendermint codebase.
We provide a `third_party.sh` script that does most of the heavy lifting:

```shell
./third_party.sh

# Instructions below based on the guide at: https://github.com/tendermint/tendermint/blob/master/docs/tools/docker-compose.md
cd third_party/tendermint-pct-instrumentation
# Build the linux binary in ./build
make build-linux
# Build tendermint/localnode image
make build-docker-localnode
```

## Quick tests
To verify the setup is functional, run:

```shell
./orchestrate.py quick_tests
```

The command outputs logs of 4 tests to the terminal.
If all is well, the script should terminate successfully within 15 minutes.

## Reproducing the results from the paper
### Baseline
Run the baseline orchestration script to begin fuzzing with the baseline implementation:

```shell
./baseline.py
```

This will execute 200 runs that randomly drop or corrupt (bit-wise) messages.
Logs of consensus messages exchanged are saved into `baseline_logs/`, and after each test run, we print the number of passed and failed tests.
A test fails when the cluster does not commit a new transaction within 1 minute after ceasing all drops and corruptions ('healing' the network), indicating a *termination violation*.

### Small-scope
The `reproduce` subcommand generates 200 test configurations each for different numbers of drops (0-2) and corruptions (0-2), then runs them.
The test harness is not fully hermetic and can be flaky at times, so we 'deflake' tests by rerunning failing tests up to 4 times.
As the tests are running, results are continuously uploaded to `logs_small_scope/`.
Aside from raw test event logs, the directory also contains `test_results.`sqlite3`, an SQLite database with the configurations, and pass/fail counts for all tests, which may be queried as the tests are running.
Start the orchestration script:

```shell
./orchestrate.py --scope small reproduce
```

### Any-scope
Similar to the small-scope tests, run:

```shell
./orchestrate.py --scope any reproduce
```

Test results are written to `logs_any_scope/`.

### Checking for violations
The SQLite database records termination violations.
To count test cases that consistently failed in all runs (the results presented in the paper), first, open the database:

```shell
sqlite3 logs_any_scope/test_results.sqlite3
# Or for small scope:
sqlite3 logs_small_scope/test_results.sqlite3
```

Then execute the following query:

```sql
SELECT
    json_array_length(json_extract(config, '$.drops')) AS drops,
    json_array_length(json_extract(config, '$.corruptions')) AS corruptions,
    SUM(CASE WHEN fail > 1 AND pass = 0 THEN 1 ELSE 0 END) AS fail_reliably
FROM TestResults
GROUP BY 1, 2
ORDER BY 1, 2;
```

The test orchestration server continuously monitors the chain for the other correct properties.
If an inconsistency is detected, the test transitions to the special `diff-commits` state, which is written to the event log for the test.
To check if this occurred in any of the executed test cases, check the logs directory for this special label:

```shell
grep -r diff-commits baseline_logs/ logs_any_scope/ logs_small_scope/ 
```

With our test setup, we have not detected any *validity*, *integrity* or *agreement*, so you should expect this to return no results.

## Adding new corruption types
If you wish to extend this codebase and add new types of structural corruptions, follow these steps:
1. In `data.py`: add your new corruption type to the `CorruptionType` enum.
2. In `data.py`: add the new enum variant to one of `ALL_PROPOSAL_CORRUPTION_TYPES`, `ALL_PROPOSAL_CORRUPTION_TYPES_ANY_SCOPE`, `ALL_VOTE_CORRUPTION_TYPES` or `ALL_VOTE_CORRUPTION_TYPES_ANY_SCOPE`.
3. In `byzzfuzz/instance.go`: put the new corruption type with the same value as the python version.
4. In `byzzfuzz/corruption.go`: add a case to the `Action` function on `MessageCorruption` to apply the new mutation. 
