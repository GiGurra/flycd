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

# Check that no git tag exists for the current commit
if [[ $(git tag --points-at HEAD) ]]; then
  echo "Tag for the current commit already exists"
  exit 1
fi

# Parse the tag from main.go
TAG=$(cat main.go | grep -oP '(?<=Version = ").*(?=")')

# Check that the tag follows semver
if [[ ! $TAG =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Tag does not follow semver (e.g. v1.2.3)"
  exit 1
fi

echo "Tagging commit with $TAG"

# Tag the commit
git tag $TAG

# Push the commit and tag to origin
git push --tags origin master



