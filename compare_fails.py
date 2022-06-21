#!/usr/bin/env python3
import os
from check_ok import check_ok

drop1 = set()
for d in os.listdir("drop1/"):
    with open(f"drop1/{d}", "r") as f:
        if not check_ok(f):
            drop1.add(d)

drop1_5m = set()
for d in os.listdir("drop1_5m/"):
    with open(f"drop1_5m/{d}", "r") as f:
        if not check_ok(f):
            drop1_5m.add(d)

print(f"drop1: {len(drop1)}")
print(f"drop1_5m: {len(drop1_5m)}")

print("In drop1 but not in drop1_5m:")
cnt = 0
for d in sorted(drop1 - drop1_5m):
    cnt += 1
    print(d)
print(cnt)

print("In drop1_5m but not in drop1:")
cnt = 0
for d in sorted(drop1_5m - drop1):
    cnt += 1
    print(d)
print(cnt)

print("In both")
cnt = 0
for d in sorted(drop1_5m & drop1):
    cnt += 1
    print(d)
print(cnt)