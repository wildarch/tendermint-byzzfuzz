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
curl -Lo $ZIPFILE https://github.com/netrixframework/tendermint-testing/archive/ef12041869655577851f603aee1eb8a9b756f089.zip
unzip $ZIPFILE
rm -f $ZIPFILE
mv tendermint-testing-* tendermint-testing/

# tendermint-test
ZIPFILE=tendermint-test.zip
rm -rf tendermint-test/ $ZIPFILE 
curl -Lo $ZIPFILE https://github.com/ImperiumProject/tendermint-test/archive/89de7d0d2208568d5e70d42b4d85986c669b4df4.zip
unzip $ZIPFILE
rm -f $ZIPFILE
mv tendermint-test-* tendermint-test/