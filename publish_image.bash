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

# Check that we have a git tag for the current commit
if [[ -z $(git tag --points-at HEAD) ]]; then
  echo "Tag the current commit first"
  exit 1
fi

# Check that the current commit is tagged with a SemVer tag
if [[ ! $(git tag --points-at HEAD) =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Tag the current commit with a SemVer tag first"
  exit 1
fi

# Check that the current commit is tagged with a SemVer tag that matches the version in main.go
if [[ $(git tag --points-at HEAD) != $(grep -oP '(?<=Version = ").*(?=")' main.go) ]]; then
  echo "Tag the current commit with a SemVer tag that matches the version in main.go"
  exit 1
fi



