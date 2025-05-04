#!/usr/bin/env sh

set -o errexit
set -o nounset

GOFLAGS="-buildvcs=false" golangci-lint run
