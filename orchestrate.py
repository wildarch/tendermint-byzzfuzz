#!/usr/bin/env python3
from dataclasses import dataclass
import dataclasses
import itertools
import subprocess
import json
import sys
import os

@dataclass(eq=True, order=True)
class MessageDrop:
    step: int
    partition: list[list[int]]

@dataclass(eq=True, order=True)
class MessageCorruption:
    step: int
    from_node: int
    to_node: int
    corruption: int

@dataclass(eq=True)
class ByzzFuzzInstanceConfig:
    drops: MessageDrop
    corruptions: MessageCorruption


ALL_PARTITIONS = [
	# 1 partition (disabled because it is equivalent to having no partition at all)
	# [[0, 1, 2, 3]],

	# 2 partitions
	[[0, 1], [2, 3]],
	[[0, 2], [1, 3]],
	[[0, 3], [1, 2]],
	[[0], [1, 2, 3]],
	[[1], [0, 2, 3]],
	[[2], [0, 1, 3]],
	[[3], [0, 1, 2]],

	# 3 partitions
	[[0], [1], [2, 3]],
	[[0], [2], [1, 3]],
	[[0], [3], [1, 2]],
	[[1], [2], [0, 3]],
	[[1], [3], [0, 2]],
	[[2], [3], [0, 1]],

	# 4 partitions
	[[0], [1], [2], [3]],
]

MAX_STEPS = 10
ALL_DROPS = [MessageDrop(step, part) for step, part in itertools.product(range(MAX_STEPS), ALL_PARTITIONS)]

def run_instance(config, liveness_timeout="1m"):
    proc = subprocess.Popen(["go", "run", "./cmd/server.go", "run-instance", f"--liveness-timeout={liveness_timeout}"], stdin=subprocess.PIPE, stderr=subprocess.PIPE)
    js = json.dumps(dataclasses.asdict(config))
    proc.stdin.write(bytes(js, "utf-8"))
    proc.stdin.flush()

    print("Process starts")
    events = []
    for line in iter(proc.stderr.readline,''):
        line = line.decode("utf-8")
        sys.stdout.write(line)

        event = json.loads(line)
        events.append(event)
        if "msg" in event and event["msg"] == "Testcase succeeded" or event["msg"] == "Testcase failed":
            try:
                proc.wait(30)
            except subprocess.TimeoutExpired:
                print("WARN: Timeout expired, terminating")
                proc.terminate()
            break
    return events

# 1m
for i, drop in enumerate(ALL_DROPS):
    logpath = f"drop1/events{i:03}.log"
    if os.path.isfile(logpath):
        print("Skip already processed: ", drop)
        continue

    inst = ByzzFuzzInstanceConfig([drop], []) 
    print(f"Run instance: {inst}")
    events = run_instance(inst)

    with open(logpath, 'w') as logfile:
        for e in events:
            json.dump(e, logfile)
            logfile.write('\n')

# HACK 5M
for i, drop in enumerate(ALL_DROPS):
    logpath = f"drop1_5m/events{i:03}.log"
    if os.path.isfile(logpath):
        print("Skip already processed: ", drop)
        continue

    inst = ByzzFuzzInstanceConfig([drop], []) 
    print(f"Run instance: {inst}")
    events = run_instance(inst, liveness_timeout="5m")

    with open(logpath, 'w') as logfile:
        for e in events:
            json.dump(e, logfile)
            logfile.write('\n')