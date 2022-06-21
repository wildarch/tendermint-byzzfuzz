#!/usr/bin/python3
import argparse
import json

def check_ok(f):
    success = None
    for line in f:
        e = json.loads(line)
        if "msg" in e and e["msg"] == "Testcase succeeded":
            success = True
        elif "msg" in e and e["msg"] == "Testcase failed":
            success = False
    return success

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('logfiles', type=argparse.FileType('r'), nargs='+')
    args = parser.parse_args()

    for logfile in args.logfiles:
        success = check_ok(logfile)
        print(f"{logfile.name}: {success}")