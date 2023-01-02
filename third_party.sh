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

# Patch bridge address of host (assumes the host IP on the bridge is 192.167.0.1)
sed -i 's/10.0.0.8/192.167.0.1/g' tendermint-pct-instrumentation/networks/local/localnode/config-template.toml
# Have the tendermint nodes create empty blocks.
# This provides a steady flow of messages to drop/corrupt.
sed -i 's/^create-empty-blocks = false/create-empty-blocks = true\ntimeout-commit = "10s"/g' tendermint-pct-instrumentation/networks/local/localnode/config-template.toml
# Fix for permissions errors, because docker runs nodes as root
sed -i 's/rm -rf \$(BUILDDIR)\/node\*/docker run --rm -v $(BUILDDIR):\/tendermint alpine rm -rf \/tendermint\/node0 \/tendermint\/node1 \/tendermint\/node2 \/tendermint\/node3/' tendermint-pct-instrumentation/Makefile