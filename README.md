# Getting started
## Requirements
- Golang 1.18
- Docker
- [docker-compose](https://pypi.org/project/docker-compose/). Not the one built-in to recent versions of docker. I found pip to be the easiest way to install it: `pip3 install --user docker-compose`.

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

## Types of violations
Our Tendermint test setup has only found *termination violations*, and thus the next few sections describe only how to find those.
After we have covered how to run the different configurations, we discuss how to check that there were indeed no *validity*, *integrity* or *agreement* violations in any of the runs.

## Running baseline
Run the baseline orchestration script to begin fuzzing with the baseline implementation:

```shell
./baseline.py
```

This will execute 200 runs that randomly drop or corrupt (bit-wise) messages.
Logs of consensus messages exchanged are saved into `baseline_logs/`, and after each test run, we print the number of passed and failed tests.
A test fails when the cluster does not commit a new transaction within 1 minute after ceasing all drops and corruptions ('healing' the network), indicating a *termination violation*.

## Running small-scope
```shell
./orchestrate.py --scope small fuzz-deflake --max-drops 2 --max-corruptions 2
```

## Running any-scope
```shell
./orchestrate.py --scope any fuzz-deflake --max-drops 2 --max-corruptions 2
```

## Validity
A correct process may only decide a value that was proposed by a correct process.

## Agreement
Our test harness implements agreement checking by keeping track of the block IDs that nodes commit. 
If a node commits a block id that does not correspond to what another node has already committed for that height, the test transitions to the special 'diff-commits' label, which will appear in the logs.
To check that this state was not reached in any of the logged runs, run:

```shell
grep -r diff-commits baseline_logs/ logs_any_scope/ logs_small_scope/ 
```

This should return no results.

## Integrity
The agreement test also covers integrity: if a node first commits one block, then commits another different block, the orchestration server detects that two different block IDs have been committed, and assigns the same 'diff-commits' label.