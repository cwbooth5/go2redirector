#!/bin/bash

if [ -z "$1" ]
then
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out
else
    #go test -v -run $1
    go test -v -timeout 30s -run ^$1$ github.com/cwbooth5/go2redirector/core
fi

