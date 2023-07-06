#!/usr/bin/env bash

set -e

cat app.yaml | yq | jq '."fly.toml".overwrite' | yj -jt > fly.toml

fly launch --ha=false --auto-confirm --now --copy-config --name=project1-app1-foobar12341



