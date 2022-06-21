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
        if "msg" in event and (event["msg"] == "Testcase succeeded" or event["msg"] == "Testcase failed"):
            try:
                proc.wait(30)
            except subprocess.TimeoutExpired:
                print("WARN: Timeout expired, terminating")
                proc.terminate()
            break
    return events

def drop1_all():
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

def drop1_5m_all():
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

def random_config(nrof_drops=1, nrof_corruptions=0):
    drops = sorted(random.sample(ALL_DROPS, nrof_drops))

    # Faulty node is fixed throughout execution
    faulty = random.randint(0, 4)

    corruptions = []
    for _ in range(nrof_corruptions):
        step = random.randint(0, MAX_STEPS)
        if step % 3 == 0:
            # Proposal
            all_corruption_types = ALL_PROPOSAL_CORRUPTION_TYPES
        else:
            # Vote (prevote, precommit)
            all_corruption_types = ALL_VOTE_CORRUPTION_TYPES
        corruption = random.choice(all_corruption_types)
        corruption = MessageCorruption(
            step = step,
            from_node = faulty,  
            to_nodes = random.choice(ALL_SUBSETS),
            corruption_type = corruption, 
        )
        corruptions.append(corruption)

    return ByzzFuzzInstanceConfig(drops, corruptions)

def dump_events(path, events):
    with open(path, 'w') as logfile:
        for e in events:
            json.dump(e, logfile)
            logfile.write('\n')

def create_db():
    conn = sqlite3.connect("logs/test_results.sqlite3", isolation_level=None)
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
    conn = create_db()
    cur = conn.cursor()

    while True:
        fuzz_one(cur, args)

def fuzz_one(cur, args):
    config = random_config(args.drops, args.corruptions)
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

    dump_events(f"logs/events{rowid:06}.log", events)

def parse_config(json_config):
    c = json.loads(json_config)
    return ByzzFuzzInstanceConfig(
        [MessageDrop(**d) for d in c["drops"]],
        [MessageCorruption(**c) for c in c["corruptions"]]
    )

def deflake(args):
    conn = create_db()
    cur = conn.cursor()

    while True:
        deflake_one(cur, args)

def deflake_one(cur, args):
    cur.execute("select rowid, config from TestResults ORDER BY pass+fail ASC LIMIT 1")
    rowid, json_config = cur.fetchone()
    config = parse_config(json_config)
    print(config)

    events = run_instance(config)

    if check_ok(events):
        cur.execute("UPDATE TestResults SET pass = pass + 1 WHERE rowid=?", (rowid,))
        dump_events(f"logs/events{rowid:06}_pass.log", events)
    else:
        cur.execute("UPDATE TestResults SET fail = fail + 1 WHERE rowid=?", (rowid,))
        dump_events(f"logs/events{rowid:06}_fail.log", events)

def fuzz_deflake(args):
    conn = create_db()
    cur = conn.cursor()

    while True:
        print("=== FUZZ ===")
        fuzz_one(cur, args)
        print("=== DEFLAKE ===")
        deflake_one(cur, args)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers()
    subparsers.required = True
    subparsers.dest = "commmand"

    parser_fuzz = subparsers.add_parser("fuzz")
    parser_fuzz.set_defaults(func=fuzz)
    parser_fuzz.add_argument("--drops", type=int, default=1)
    parser_fuzz.add_argument("--corruptions", type=int, default=0)

    parser_deflake = subparsers.add_parser("deflake")
    parser_deflake.set_defaults(func=deflake)

    parser_fuzz_deflake = subparsers.add_parser("fuzz-deflake")
    parser_fuzz_deflake.set_defaults(func=fuzz_deflake)
    parser_fuzz_deflake.add_argument("--drops", type=int, default=1)
    parser_fuzz_deflake.add_argument("--corruptions", type=int, default=0)

    args = parser.parse_args()
    args.func(args)