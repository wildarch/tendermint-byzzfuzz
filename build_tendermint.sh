#!/bin/bash
mkdir -p build/
cd third_party/tendermint-pct-instrumentation
go build -o ../../build/ ./cmd/tendermint