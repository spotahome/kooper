#!/usr/bin/env sh

set -o errexit
set -o nounset

go generate ./...
mockery
