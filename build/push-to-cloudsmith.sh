#!/bin/bash
releaseVersion=$(echo "$RELEASE" | tr '/' -)
releaseName="sa-collector-$releaseVersion-$GOOS-$GOARCH.tar.gz"
go build -ldflags "-s -w" -o bin && tar -czvf "$releaseName" bin

# cloudsmith push raw vinted/raw-hosted-security "$releaseName"