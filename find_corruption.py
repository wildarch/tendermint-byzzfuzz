#!/usr/bin/env python3
from data import *
import dataclasses
from orchestrate import create_db, run_instance, parse_config
import json

conn = create_db()
cur = conn.cursor()
# A reliably failing config with one corruption
cur.execute("select config FROM TestResults WHERE pass = 0 AND fail = 5 AND json_array_length(json_extract(config, '$.corruptions')) = 1")
for json_config, in cur.fetchall():
    config = parse_config(json_config)
    new_config = ByzzFuzzInstanceConfig(config.drops, [])

    new_json_config = json.dumps(dataclasses.asdict(new_config))
    cur.execute("select rowid, pass, fail FROM TestResults WHERE config = ?", (new_json_config,))
    res = cur.fetchone()
    if res is None:
        print("Config not yet tested: ", new_json_config)
    elif res[1] == 0:
        print("Config also fails without corruption")
    else:
        print("Found!", res)
