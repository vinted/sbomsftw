#!/bin/bash
releaseVersion=$(echo "$RELEASE" | tr '/' -)
releaseName="sa-collector-$releaseVersion-$GOOS-$GOARCH.tar.gz"
go build -ldflags "-s -w" -o sa-collector && tar -czvf "$releaseName" sa-collector

pip3 install cloudsmith-cli
cloudsmith push raw vinted/raw-hosted-security "$releaseName"
