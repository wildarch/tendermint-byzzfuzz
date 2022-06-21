#!/usr/bin/env python3
import json
from dataclasses import dataclass
import tkinter as tk
import subprocess
import argparse
from data import *
import re

@dataclass(eq=True, frozen=True)
class Event:
    is_receive: bool
    is_send: bool
    sent_from: str
    sent_to: str
    msg_type: str
    height: int
    round: int

parser = argparse.ArgumentParser()
parser.add_argument('logfile', type=argparse.FileType('r'))
args = parser.parse_args()

match = re.match(r".*events(\d\d\d).log", args.logfile.name)
if match is not None:
    num = int(match.group(1))
    print(ALL_DROPS[num])

retransmitted_events = set()
events = set()

for event in args.logfile:
    try:
        e = json.loads(event)
        if "drops" in e:
            print(e)
        if "msg" in e and e["msg"] == "Consensus message":
            event = Event(
                e["is_receive"], 
                e["is_send"], 
                e["sent_from"], 
                e["sent_to"], 
                e["type"], 
                e["height"], 
                e["round"],
            )
            if event in events:
                retransmitted_events.add(event)
            else:
                events.add(event)
    except json.JSONDecodeError:
        print("Cannot parse line: ", event)

rounds = set()
for e in events:
    rounds.add((e.height, e.round))

rounds = sorted(rounds)

print(rounds)

def is_received(event):
    assert event.is_send

    return Event(
        is_receive=True, 
        is_send=False, 
        sent_from=event.sent_from, 
        sent_to=event.sent_to, 
        msg_type=event.msg_type, 
        height=event.height, 
        round=event.round,
    ) in events

for (height, round) in rounds:
    for step in ["Proposal", "Prevote", "Precommit"]:
        print(f"H={height}/R={round}/S={step}")

        for event in events:
            if event.is_receive or event.msg_type != step or event.height != height or event.round != round:
                continue
            if is_received(event):
                print("OK: ", event)
            else:
                print("DROP: ", event)

NODE_HEIGHT = {
    "node0": 50,
    "node1": 100,
    "node2": 150,
    "node3": 200,
}

window = tk.Tk()
window.geometry("2400x300")

canvas = tk.Canvas(window, width=2400, height=300)
canvas.pack()

x_off = 0
for (height, round) in rounds:
    canvas.create_text(x_off+150, 240, text=f"H={height}/R={round}")
    for step in ["Proposal", "Prevote", "Precommit"]:
        canvas.create_text(x_off+50, 220, text=step)

        for event in events:
            if event.is_receive or event.msg_type != step or event.height != height or event.round != round:
                continue
            if is_received(event):
                canvas.create_line(x_off, NODE_HEIGHT[event.sent_from], x_off+100, NODE_HEIGHT[event.sent_to], arrow=tk.LAST, width=3)
            else:
                canvas.create_line(x_off, NODE_HEIGHT[event.sent_from], x_off+100, NODE_HEIGHT[event.sent_to], arrow=tk.LAST, dash=(5,2))
        x_off += 100
    x_off += 50

canvas.update()
canvas.postscript(file="diagram.ps")
subprocess.run(["ps2pdf", "-dEPSCrop", "diagram.ps"])

window.mainloop()