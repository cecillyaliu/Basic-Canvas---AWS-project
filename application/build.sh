#!/bin/bash
rm -rf output
set -e

GOARCH=amd64 GOOS=linux go build .
mkdir output
cp demo output/demo
cp scripts/bootstrap.sh output/bootstrap.sh
chmod +x output/bootstrap.sh
cp -r opt output