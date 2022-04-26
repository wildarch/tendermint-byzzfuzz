#!/bin/bash

mkdir -p vendor/
cd vendor/

# pct-instrumentation
ZIPFILE=pct-instrumentation.zip
rm -rf tendermint-pct-instrumentation/ $ZIPFILE 
curl -LO https://github.com/zeu5/tendermint/archive/refs/heads/$ZIPFILE
unzip $ZIPFILE
rm -f $ZIPFILE

# tendermint-test
ZIPFILE=tendermint-test.zip
rm -rf tendermint-test/ $ZIPFILE 
curl -Lo $ZIPFILE https://github.com/ImperiumProject/tendermint-test/archive/89de7d0d2208568d5e70d42b4d85986c669b4df4.zip
unzip $ZIPFILE
rm -f $ZIPFILE
mv tendermint-test-* tendermint-test/