#!/usr/bin/python3
import argparse
import json

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('logfiles', type=argparse.FileType('r'), nargs='+')
    args = parser.parse_args()

    for logfile in args.logfiles:
        success = None
        for line in logfile:
            e = json.loads(line)
            if "msg" in e and e["msg"] == "Testcase succeeded":
                success = True
            elif "msg" in e and e["msg"] == "Testcase failed":
                success = False
        print(f"{logfile.name}: {success}")