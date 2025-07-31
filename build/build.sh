#!/bin/bash
CGO_ENABLED=1 go build -tags "sqlite_json sqlite_fts5 sqlite_math_functions sqlite3" -ldflags "-s -w" -o sa-collector
