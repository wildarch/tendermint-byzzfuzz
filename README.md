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

## Running baseline
Run the baseline orchestration script to begin fuzzing with the baseline implementation:

```shell
./baseline.py
```

This will execute 200 runs that randomly drop or corrupt (bit-wise) messages.
Logs of consensus messages exchanged are saved into `baseline_logs/`, and after each test run, we print the number of passed and failed tests.
In this configuration (nearly) all tests should be successful.

## Running small-scope






## Running
First start server:
```
cd third_party/tendermint-testing
go run ./server.go
```

If you see errors about failing to bind to the address, it is possible the bridge address has not yet been created.
In this case you can start the nodes first one time (see below) and stop them with Ctrl-C once they have started. This should create the bridge.

**The server must start before the nodes, or the nodes will not connect to the server.**

Now start nodes:
```
cd third_party/tendermint-pct-instrumentation
make localnet-start
```

In the testing server logs you should start to see JSON messages. Look for `Starting testcase`, and eventually you should see `Testcase succeeded`.

If you see permission errors when starting the network for the second time change this in `third_party/tendermint-pct-instrumentation/Makefile`:

```diff
    # Stop testnet
    localnet-stop:
    docker-compose down
-   rm -rf $(BUILDDIR)/node*
+   docker run --rm -v $(BUILDDIR):/tendermint alpine rm -rf /tendermint/node{0,1,2,3}
    .PHONY: localnet-stop
```


# To add to docs
https://docs.docker.com/network/bridge/#enable-forwarding-from-docker-containers-to-the-outside-world

docker-compose up --no-start