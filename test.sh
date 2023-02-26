#!/usr/bin/env bash

set -e

# Functional testing with race detection
go test -v -race ./...

# Fuzzing
pushd core
go test -v -fuzz FuzzMakeNewKeyword -fuzztime 10000x
go test -v -fuzz FuzzMakeNewlink -fuzztime 10000x
go test -v -fuzz FuzzCreateStringVar -fuzztime 10000x
go test -v -fuzz FuzzCreateMapVar -fuzztime 10000x
go test -v -fuzz FuzzParsePath -fuzztime 10000x
go test -v -fuzz FuzzSanitizeURL -fuzztime 10000x
go test -v -fuzz FuzzGetLink -fuzztime 10000x
popd
pushd http
go test -v -fuzz FuzzRouteLogin -fuzztime 10000x
popd
