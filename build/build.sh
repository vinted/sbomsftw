#!/bin/bash
env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o sa-collector
