#!/bin/bash
set -e
export GOROOT=~/.goenv/versions/1.23.9
export PATH=$GOROOT/bin:$PATH
export GOSUMDB=off

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

echo "Building fnsqldb ${VERSION} for Linux amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o fnsqldb .
ls -lh fnsqldb
echo "Done: ${VERSION}"
