#!/bin/bash
TEST=crashreplica

mkdir -p node_homes/
rm -rf node_homes/*

cp -r vendor/tendermint-test/logs/$TEST/* node_homes/

./patch_node_config.py