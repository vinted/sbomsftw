#!/bin/bash
env GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o sa-collector
