#!/bin/bash
releaseVersion=$(echo "$RELEASE" | tr '/' -)
releaseName="sa-collector-$releaseVersion-$GOOS-$GOARCH"

CGO_ENABLED=0 go build -ldflags "-s -w" -o "$releaseName"

pip3 install cloudsmith-cli
cloudsmith push raw "$CS_REPOSITORY" "$releaseName"
