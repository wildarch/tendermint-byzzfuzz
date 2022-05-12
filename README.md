# Getting started
## Requirements
- Golang 1.18
- Docker
- [docker-compose](https://pypi.org/project/docker-compose/). Not the one built-in to recent versions of docker. I found pip to be the easiest way to install it: `pip3 install --user docker-compose`.

## Setup
```
./third_party.sh

# Patch bridge address of host (assumes the host IP on the bridge is 192.167.0.1)
./patch.sh

# Following the guide at https://github.com/tendermint/tendermint/blob/master/docs/tools/docker-compose.md
cd third_party/tendermint-pct-instrumentation

# Build the linux binary in ./build
make build-linux

# (optionally) Build tendermint/localnode image
make build-docker-localnode
```

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

You in the testing server logs you should start to see JSON messages. Look for `Starting testcase`, and eventually ou should see `Testcase succeeded`.

If you see permission errors when starting the network for the second time change this in `third_party/tendermint-pct-instrumentation/Makefile`:

```diff
    # Stop testnet
    localnet-stop:
    docker-compose down
-   rm -rf $(BUILDDIR)/node*
+   docker run --rm -v $(BUILDDIR):/tendermint alpine rm -rf /tendermint/node{0,1,2,3}
    .PHONY: localnet-stop
```