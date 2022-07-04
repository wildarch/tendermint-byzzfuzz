#!/usr/bin/env python3
import sqlite3
import shutil

shutil.copyfile("logs/test_results.sqlite3", "logs/test_results_final.sqlite3")
conn = sqlite3.connect("logs/test_results_final.sqlite3", isolation_level=None)
cur = conn.cursor()

cur.execute("ALTER TABLE TestResults ADD COLUMN is_final BOOL")

for d in range(3):
  for c in range(3):
    if d == 0 and c == 0:
      continue
    cur.execute("""
        SELECT rowid 
        FROM TestResults 
        WHERE json_array_length(json_extract(config, '$.drops'))  = ?
          AND json_array_length(json_extract(config, '$.corruptions'))  = ?
        ORDER BY rowid
        LIMIT 200
    """, (d, c))

    rowids = [r[0] for r in cur.fetchall()]
    assert len(rowids) == 200

    cur.execute(f"""
      UPDATE TestResults
      SET is_final = TRUE
      WHERE rowid IN ({','.join(["?"]*len(rowids))})
    """, rowids)
