#!/usr/bin/env python3
import json
import sys

NODES = [
    "node0",
    "node1",
    "node2",
    "node3",
]
# One byzantine node
FAULTY = 1

# Find the expected steps
def find_expected_steps(events, node):
    height_round_received = {}
    for e in events:
        if "To" not in e or e["To"] != node:
            continue
        h = e["Height"]
        r = e["Round"]

        if (h,r) in height_round_received:
            height_round_received[(h,r)].add(e["From"])
        else:
            height_round_received[(h,r)] = set([e["From"]])

    # Expect a step if we received a message with a given height,round from > f+1 nodes
    return frozenset([hr for hr,nodes in height_round_received.items() if len(nodes) > FAULTY+1])

# Now check which steps actually occurred
def find_actual_steps(events, node):
    actual_steps = set()
    for e in events:
        if "Replica" not in e or e["Replica"] != node:
            continue

        h = e["Height"]
        r = e["Round"]

        actual_steps.add((h,r))
    return frozenset(actual_steps)

if __name__ == "__main__":
    # Read events
    f = open('spec.log')
    events = []
    for line in f:
        e = json.loads(line)
        events.append(e)

    f.close()

    found_error = False
    for node in NODES:
        expected_steps = find_expected_steps(events, node)
        actual_steps = find_actual_steps(events, node)

        # Check for missing steps
        missing_steps = expected_steps - actual_steps
        if len(missing_steps) > 0:
            print(f"[{node}] missing steps: {sorted(missing_steps)}")
            found_error = True
        else:
            print(f"[{node}] spec check OK")
    
    if found_error:
        print("Errors found.")
        sys.exit(1)
    else:
        print("All good.")
