#!/usr/bin/env bash

set -e

rm -rf mocks && go clean ./... && mockery && go build ./... && go test ./...