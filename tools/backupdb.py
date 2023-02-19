#!/usr/bin/env python3

"""
Remotely back up the go2redirector database.

This is useful for maintaining an off-box backup.
"""

from datetime import date
import json
import sys
import urllib.request

# redirector URL (including port, if nonstandard)
try:
	target = sys.argv[1]
except IndexError:
	print("specify the HTTP/HTTPS URL for the redirector to back up")
	sys.exit(1)

today = date.today()
backupdate = today.strftime("%m-%d-%y")
backupfile = f'{backupdate}-go2backup.json'

print(f"running backup of remote go2redirector db to: {backupfile}")

resp = urllib.request.urlopen(f"{target}/_db_")
data = resp.read().decode('utf-8')

stats = json.loads(data)
print(f"Links: {len(stats['Lists'])}, Lists: {len(stats['Links'])}")

with open(backupfile, 'w') as f:
	f.write(data)

print("done")

