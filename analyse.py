#!/usr/bin/env python3
import json

f = open('spec.log')

events = []
for line in f:
    e = json.loads(line)
    events.append(e)

f.close()


for e in events:
    if "Replica" in e and e["Replica"] == "node0":
        print(e)