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


if __name__ == "__main__":
    conn = create_db()
    cur = conn.cursor()

    while True:
        config = random_config(1, 1)
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

    conn.close()
    