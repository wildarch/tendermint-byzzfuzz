#!/bin/bash
set -e

sed -i 's/10.0.0.8/192.167.0.1/g' third_party/tendermint-pct-instrumentation/networks/local/localnode/config-template.toml
sed -i 's/172.23.37.208/192.167.0.1/g' third_party/tendermint-testing/server.go
sed -i 's/10.0.0.8/192.167.0.1/g' third_party/tendermint-test/server.go