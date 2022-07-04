#!/bin/bash
set -e

rsync -zrt munich:tendermint-byzzfuzz/logs/ logs/
