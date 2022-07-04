#!/bin/bash
set -e

mkdir -p third_party/
rm -rf third_party/*
cd third_party/

# pct-instrumentation
ZIPFILE=pct-instrumentation.zip
rm -rf tendermint-pct-instrumentation/ $ZIPFILE 
curl -LO https://github.com/zeu5/tendermint/archive/refs/heads/$ZIPFILE
unzip $ZIPFILE
rm -f $ZIPFILE

sed -i 's/10.0.0.8/192.167.0.1/g' tendermint-pct-instrumentation/networks/local/localnode/config-template.toml
sed -i 's/^create-empty-blocks = false/create-empty-blocks = true\ntimeout-commit = "10s"/g' tendermint-pct-instrumentation/networks/local/localnode/config-template.toml