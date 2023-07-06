#!/usr/bin/env bash

set -e

# Check that everything is committed to git
if [[ -n $(git status -s) ]]; then
  echo "Commit everything to git first"
  exit 1
fi

# Check that the current branch is master
if [[ $(git rev-parse --abbrev-ref HEAD) != "master" ]]; then
  echo "Switch to master branch first"
  exit 1
fi

# Check that the current branch is up to date with origin
if [[ $(git rev-parse HEAD) != $(git rev-parse @{u}) ]]; then
  echo "Pull from origin first"
  exit 1
fi

