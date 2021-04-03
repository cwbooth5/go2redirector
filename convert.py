#!/usr/bin/env python3

<<<<<<< HEAD
=======
"""
This script exists for conversion of godb.json from the previous version to the current version.

It should be run if json marshalling errors show up when attempting to load the DB.
"""

>>>>>>> 15-tag-improvements
import os
import json

with open("godb.json") as f:
    data = json.loads(f.read())

for lst, contents in data['Lists'].items():
    for linkid, tag in contents['TagBindings'].items():
        if type(tag) == str:
            print(f"List [{lst}] linkid [{linkid}] tag [{tag}] converted to list")
            contents['TagBindings'][linkid] = [tag]

os.rename("godb.json", "godb.json.backup")

with open("godb.json", "w") as f:
    f.write(json.dumps(data))

print("conversion complete")
