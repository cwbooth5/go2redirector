#!/usr/bin/env python3

"""
This script exists for conversion of godb.json from the previous version to the current version.

It should be run if json marshalling errors show up when attempting to load the DB.
"""

import os
import json
import sys


def main():
    original_file = sys.argv[1]
    with open(original_file) as f:
        data = json.loads(f.read())

    for lst, contents in data['Lists'].items():
        for linkid, tag in contents['TagBindings'].items():
            if type(tag) == str:
                print(f"List [{lst}] linkid [{linkid}] tag [{tag}] converted to list")
                contents['TagBindings'][linkid] = [tag]

    os.rename(original_file, f"{original_file}.backup")

    with open(original_file, "w") as f:
        f.write(json.dumps(data))

    print(f"conversion complete, backup saved to {original_file}.backup")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit("enter a godb file in JSON format for conversion")
    main()
