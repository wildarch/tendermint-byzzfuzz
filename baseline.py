#!/usr/bin/env python3
import subprocess
import sys
import json
import os
from pathlib import Path

def run_baseline():
    proc = subprocess.Popen(["go", "run", "./cmd/server.go", "baseline"], stdin=subprocess.PIPE, stderr=subprocess.PIPE)

    events = []
    for line in iter(proc.stderr.readline,''):
        line = line.decode("utf-8")
        sys.stdout.write(line)

        try:
            event = json.loads(line)
            events.append(event)
            if "msg" in event and (event["msg"] == "Testcase succeeded" or event["msg"] == "Testcase failed"):
                try:
                    proc.wait(30)
                except subprocess.TimeoutExpired:
                    print("WARN: Timeout expired, terminating")
                    proc.terminate()
                break
        except json.decoder.JSONDecodeError:
            print(f"WARN: cannot decode line '{line}'")
    return events

def check_ok(events):
    for e in events:
        if "msg" in e and e["msg"] == "Testcase succeeded":
            success = True
        elif "msg" in e and e["msg"] == "Testcase failed":
            success = False
    return success

if __name__ == "__main__":
    passed = 0
    failed = 0
    total = 0
    Path("baseline_logs/").mkdir(parents=True, exist_ok=True)
    for i in range(200):
        logpath = f"baseline_logs/events{i:03}.log"
        if os.path.isfile(logpath):
            print("Skip already processed: ", i)
            continue
        events = run_baseline()

        total += 1
        if check_ok(events):
            passed += 1
        else:
            failed += 1
        print(f"Passed: {passed}, Failed: {failed}, Total: {total}")

        with open(logpath, 'w') as logfile:
            for e in events:
                json.dump(e, logfile)
                logfile.write('\n')