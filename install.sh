#!/bin/bash

# This is here for two purposes.
# 1. It allows us to define our initial config JSON without tracking the config file itself.
# 2. It allows comments so users know what fields mean.

GOCONFIG="go2config.json"
GODB="godb.json"

JSON=$(grep -v '^ *#' << EOF
{
    "_comment": "This is a config file for the go2 redirector.",
# This will be the address the web server listens on locally.
    "local_listen_address": "127.0.0.1",
# This is the TCP port which the above address will open.
    "local_listen_port": 8080,
# If behind a NAT or load balancer, this is the external IP clients see.
    "external_address": "localhost",
# If the redirector is behind an NAT or load balancer, this would be the client-facing port.
    "external_port": 8080,
    "godb_filename": "godb.json",
# The redirector name in both the go2/ redirects themselves and the HTML templates
    "redirector_name": "go2",
# The time interval for the redirector to prune links
    "prune_interval": "1m",
# The default behavior of newly-created lists of links (rList, rTop, rFreshest, rRandom)
    "new_list_behavior": "rFreshest",
# Levenshtein distance ratio - this is used to find similar keywords and it is an edit distance metric.
# The valid values are between .1 and 1.0. Lower numbers mean the matching is more generous.
# Higher ratios mean it matches less.
    "levenshtein_distance_ratio": 0.6,
# This boolean controls whether new keywords have link logging
# enabled by default.
    "link_log_new_keywords": true,
# A given special keyword can have logging enabled and the capacity of the log is determined by this variable.
# This controls the number of recent usages shown for any given special redirects.
    "link_log_capacity": 10,
# This is a path to a log file for the redirector.
    "log_file": "redirector.log"
}
EOF
)

# Yes, it's silly to force someone to have jq, but for this we use it to validate the json.
if ! command -v jq &>/dev/null; then
    echo "You need to install the 'jq' command to generate this config file.'"
    exit 1
fi

# if the file doesn't already exist, create a new one with the above contents.
if [ ! -e $GOCONFIG ]; then
    echo "Writing out new $GOCONFIG..."
    echo $JSON > $GOCONFIG
    echo "$(jq '' $GOCONFIG)" > $GOCONFIG
else
    echo "$GOCONFIG already exists, not overwriting..."
fi

# write out a godb.json file with nothing inside
if [ ! -e $GODB ]; then
    echo "Writing out new initial $GODB..."
    cp "godb.json.init" $GODB
else
    echo "$GODB already exists, not overwriting..."
fi

echo "Setup complete."
