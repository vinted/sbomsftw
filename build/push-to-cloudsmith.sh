#!/bin/bash
releaseVersion=$(echo "$RELEASE" | tr '/' -)
releaseName="sa-collector-$releaseVersion-$GOOS-$GOARCH"

go build -ldflags "-s -w" -o "$releaseName"

pip3 install cloudsmith-cli
cloudsmith push raw vinted/raw-hosted-security "$releaseName"