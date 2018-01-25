#!/usr/bin/env sh

set -o errexit
set -o nounset
set -o pipefail

go test `go list ./... | grep -v vendor` -v