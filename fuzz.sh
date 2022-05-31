#!/bin/bash
until go run ./cmd/server.go fuzz; do
        echo "Crashed, restarting in 30 seconds"
        sleep 30
done 