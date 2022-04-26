#!/bin/bash
mkdir -p build/
cd vendor/tendermint-pct-instrumentation
go build -o ../../build/ ./cmd/tendermint