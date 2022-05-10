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

# tendermint-testing
ZIPFILE=tendermint-testing.zip
rm -rf tendermint-testing/ $ZIPFILE 
curl -Lo $ZIPFILE https://github.com/netrixframework/tendermint-testing/archive/35e07a2a96ea42fc85fd16128fa5b83124d9804b.zip
unzip $ZIPFILE
rm -f $ZIPFILE
mv tendermint-testing-* tendermint-testing/