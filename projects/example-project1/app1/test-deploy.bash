#!/usr/bin/env bash

set -e

cat app.yaml | yq | jq '."fly.toml".overwrite' | yj -jt > fly.toml

fly deploy



