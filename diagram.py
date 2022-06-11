#!/usr/bin/env python3
import json
from dataclasses import dataclass
import tkinter as tk

@dataclass(eq=True, frozen=True)
class Event:
    is_receive: bool
    is_send: bool
    sent_from: str
    sent_to: str
    msg_type: str
    height: int
    round: int

events = set()

with open("server.log", "r") as logfile:
    for event in logfile:
        try:
            e = json.loads(event)
            if "msg" in e and e["msg"] == "Consensus message":
                events.add(Event(
                    e["is_receive"], 
                    e["is_send"], 
                    e["sent_from"], 
                    e["sent_to"], 
                    e["type"], 
                    e["height"], 
                    e["round"],
                ))
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
window.geometry("1200x300")

canvas = tk.Canvas(window, width=1200, height=300)
canvas.pack()

x_off = 0
for (height, round) in rounds:
    for step in ["Proposal", "Prevote", "Precommit"]:
        print(f"H={height}/R={round}/S={step}")

        for event in events:
            if event.is_receive or event.msg_type != step or event.height != height or event.round != round:
                continue
            if is_received(event):
                canvas.create_line(x_off, NODE_HEIGHT[event.sent_from], x_off+100, NODE_HEIGHT[event.sent_to], arrow=tk.LAST, width=3)
            else:
                canvas.create_line(x_off, NODE_HEIGHT[event.sent_from], x_off+100, NODE_HEIGHT[event.sent_to], arrow=tk.LAST, dash=(5,2))
        x_off += 100



window.mainloop()