#!/usr/bin/env python3
from dataclasses import dataclass
import dataclasses
import itertools
import subprocess
import json
import sys

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
#ALL_DROPS = [MessageDrop(step, part) for step, part in itertools.product(range(MAX_STEPS), ALL_PARTITIONS)]

def run_instance(config):
    proc = subprocess.Popen(["go", "run", "./cmd/server.go", "run-instance"], stdin=subprocess.PIPE, stderr=subprocess.PIPE)
    js = json.dumps(dataclasses.asdict(config))
    proc.stdin.write(bytes(js, "utf-8"))
    proc.stdin.flush()

    print("Process starts")
    for line in iter(proc.stderr.readline,''):
        line = line.decode("utf-8")
        sys.stdout.write(line)

        event = json.loads(line)
        if "msg" in event and event["msg"] == "Testcase succeeded" or event["msg"] == "Testcase failed":
            try:
                proc.wait(30)
            except subprocess.TimeoutExpired:
                print("WARN: Timeout expired, killing instead")
                proc.kill()
            break

inst = ByzzFuzzInstanceConfig([
    MessageDrop(0, [[0], [1, 2, 3]])
], [])

res = run_instance(inst)
print(res)
