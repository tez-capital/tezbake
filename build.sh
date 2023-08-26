#!/bin/bash

# PREFIX="ALPHA"

echo "Building amd64..."
if [ "$PREFIX" != "" ]; then
    env CGO_ENABLED=0 go build -tags server -ldflags="-s -w" -o "bin/$PREFIX-bb-cli-linux-amd64" 
else
    env CGO_ENABLED=0 go build -tags server -ldflags="-s -w" -o "bin/bb-cli-linux-amd64" 
fi
echo "Building arm64..."
if [ "$PREFIX" != "" ]; then
    env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags server -ldflags="-s -w" -o "bin/$PREFIX-bb-cli-linux-arm64";
else
    env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags server -ldflags="-s -w" -o "bin/bb-cli-linux-arm64";
fi
echo "Building docs..."
go run -tags server docs/build.go
