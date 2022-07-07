#!/bin/bash
set -e

rsync -zrt munich:tendermint-byzzfuzz/anyscope_logs/ anyscope_logs_remote/
