#!/usr/bin/env python3
from data import *
from dataclasses import dataclass
import dataclasses
import itertools
import subprocess
import json
import sys
import os
import random
import sqlite3
import argparse
import time
from pathlib import Path

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

def random_config(scope, nrof_drops=1, nrof_corruptions=0):
    drops = sorted(random.sample(ALL_DROPS, nrof_drops))

    # Faulty node is fixed throughout execution
    faulty = random.randrange(0, 4)

    corruptions = []
    for _ in range(nrof_corruptions):
        step = random.randrange(0, MAX_STEPS)
        if step % 3 == 0:
            # Proposal
            if scope == "small":
                all_corruption_types = ALL_PROPOSAL_CORRUPTION_TYPES
            elif scope == "any":
                all_corruption_types = ALL_PROPOSAL_CORRUPTION_TYPES_ANY_SCOPE
        else:
            # Vote (prevote, precommit)
            if scope == "small":
                all_corruption_types = ALL_VOTE_CORRUPTION_TYPES
            elif scope == "any":
                all_corruption_types = ALL_VOTE_CORRUPTION_TYPES_ANY_SCOPE
        corruption = random.choice(all_corruption_types)
        corruption = MessageCorruption(
            step = step,
            from_node = faulty,  
            to_nodes = random.choice(ALL_SUBSETS),
            corruption_type = corruption, 
            # The seed value is, among other things, used to select a previously seen message ID.
            # It should be large enough that any message in the history is reachable.
            # We cautiously pick a high value, an upper bound to the number of messages we expect 
            # the cluster to exchange in our short test period.
            seed = random.randint(0, 10_000),
        )
        corruptions.append(corruption)

    return ByzzFuzzInstanceConfig(drops, corruptions)

def dump_events(path, events):
    with open(path, 'w') as logfile:
        for e in events:
            json.dump(e, logfile)
            logfile.write('\n')

def logs_dir(args):
    return Path(f"logs_{args.scope}_scope")

def create_db(args):
    dir = logs_dir(args)
    dir.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(dir / "test_results.sqlite3", isolation_level=None)
    conn.execute('''
		CREATE TABLE IF NOT EXISTS TestResults(
			config JSON,
			pass INT,
            fail INT); 
    ''')
    return conn

def check_ok(events):
    for e in events:
        if "msg" in e and e["msg"] == "Testcase succeeded":
            success = True
        elif "msg" in e and e["msg"] == "Testcase failed":
            success = False
    return success

def fuzz(args):
    conn = create_db(args)
    cur = conn.cursor()

    while True:
        fuzz_one(cur, args)

def fuzz_one(cur, args):
    drops = args.drops
    if args.max_drops is not None:
        drops = random.randint(0, args.max_drops)
    corruptions = args.corruptions
    if args.max_corruptions is not None:
        # Do not allow 0 drops and 0 corruptions
        min_corruptions = 1 if drops == 0 else 0
        corruptions = random.randint(min_corruptions, args.max_corruptions)
    print(f"drops: {drops}, corruptions: {corruptions}")
    config = random_config(args.scope, drops, corruptions)
    events = run_instance(config)

    passed = 0
    failed = 0
    if check_ok(events):
        passed = 1
    else:
        failed = 1

    json_config = json.dumps(dataclasses.asdict(config))
    cur.execute('''
        INSERT INTO TestResults VALUES (?, ?, ?)
    ''', (json_config, passed, failed))
    rowid = cur.lastrowid

    dump_events(logs_dir(args) / f"events{rowid:06}.log", events)

def parse_config(json_config):
    c = json.loads(json_config)
    return ByzzFuzzInstanceConfig(
        [MessageDrop(**d) for d in c["drops"]],
        [MessageCorruption(**c) for c in c["corruptions"]]
    )

def deflake(args):
    conn = create_db(args)
    cur = conn.cursor()

    while True:
        deflake_one(cur, args)

def deflake_one(cur, args):
    cur.execute("select rowid, config from TestResults WHERE pass = 0 AND fail < 5 ORDER BY RANDOM() LIMIT 1")
    res = cur.fetchone()
    if res is None:
        print("WARN: Nothing to deflake")
        time.sleep(5)
        return
    rowid, json_config = res 
    config = parse_config(json_config)
    print(config)

    events = run_instance(config)

    if check_ok(events):
        cur.execute("UPDATE TestResults SET pass = pass + 1 WHERE rowid=?", (rowid,))
        dump_events(logs_dir(args) / f"events{rowid:06}_pass.log", events)
    else:
        cur.execute("UPDATE TestResults SET fail = fail + 1 WHERE rowid=?", (rowid,))
        dump_events(logs_dir(args) / f"events{rowid:06}_fail.log", events)

def fuzz_deflake(args):
    conn = create_db(args)
    cur = conn.cursor()

    while True:
        print("=== FUZZ ===")
        fuzz_one(cur, args)
        print("=== DEFLAKE ===")
        deflake_one(cur, args)

def reproduce(args):
    conn = create_db(args)
    cur = conn.cursor()

    max_drops = 2
    max_corruptions = 2
    nrof_configs = 200


    def insert_config(config):
        json_config = json.dumps(dataclasses.asdict(config))
        passed = 0
        failed = 0
        cur.execute('''
            INSERT INTO TestResults VALUES (?, ?, ?)
        ''', (json_config, passed, failed))

    # Check that we have the expected number of configs already, or nothing
    cur.execute("SELECT COUNT(*) FROM TestResults")
    count = cur.fetchone()[0]
    if count == 0:
        for d in range(max_drops+1):
            for c in range(max_corruptions+1):
                if d == 0 and c == 0:
                    continue
                
                for i in range(200):
                    insert_config(random_config(args.scope, d, c))
    elif count == 1600:
        print("Already have all configs in results table, skipping generation")
    else:
        print("Error: results table is partially populated")
        sys.exit(1)
    
    while True:
        cur.execute("select COUNT(*) FROM TestResults WHERE pass = 0 AND fail < 5")
        if cur.fetchone()[0] > 0:
            deflake_one(cur, args)
        else:
            break

    print("Done!")

def quick_tests(args):
    pass_config = ByzzFuzzInstanceConfig([], [])
    # Fully isolate all nodes
    fail_config = ByzzFuzzInstanceConfig([
        MessageDrop(
            step = 1,
            partition = [[0], [1], [2], [3]],
        )
    ], [])

    # Alternate between pass and fail tests.
    # Repeat the cycle once because the very first run starts from a clean env.
    for i in range(2):
        print("expect FAIL")
        events = run_instance(fail_config)
        assert(not check_ok(events))

        print("expect PASS")
        events = run_instance(pass_config)
        assert(check_ok(events))

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers()
    subparsers.required = True
    subparsers.dest = "commmand"

    parser_fuzz = subparsers.add_parser("fuzz")
    parser_fuzz.set_defaults(func=fuzz)
    parser_fuzz.add_argument("--drops", type=int, default=1)
    parser_fuzz.add_argument("--corruptions", type=int, default=0)
    parser_fuzz.add_argument("--max-drops", type=int)
    parser_fuzz.add_argument("--max-corruptions", type=int)
    parser_fuzz.add_argument("--scope", choices=["any", "small"], required=True)

    parser_deflake = subparsers.add_parser("deflake")
    parser_deflake.set_defaults(func=deflake)
    parser_deflake.add_argument("--scope", choices=["any", "small"], required=True)

    parser_fuzz_deflake = subparsers.add_parser("fuzz-deflake")
    parser_fuzz_deflake.set_defaults(func=fuzz_deflake)
    parser_fuzz_deflake.add_argument("--drops", type=int, default=1)
    parser_fuzz_deflake.add_argument("--corruptions", type=int, default=0)
    parser_fuzz_deflake.add_argument("--max-drops", type=int)
    parser_fuzz_deflake.add_argument("--max-corruptions", type=int)
    parser_fuzz_deflake.add_argument("--scope", choices=["any", "small"], required=True)

    parser_reproduce = subparsers.add_parser("reproduce")
    parser_reproduce.set_defaults(func=reproduce)
    parser_reproduce.add_argument("--scope", choices=["any", "small"], required=True)

    parser_quick_tests = subparsers.add_parser("quick_tests")
    parser_quick_tests.set_defaults(func=quick_tests)

    args = parser.parse_args()
    args.func(args)
