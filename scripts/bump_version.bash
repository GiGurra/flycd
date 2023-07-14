#!/usr/bin/env bash

set -e

# check that the build works
go clean ./...
go build ./...
go test ./...
go clean ./...

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
# shellcheck disable=SC1083
if [[ $(git rev-parse HEAD) != $(git rev-parse @{u}) ]]; then
  echo "Pull from origin first"
  exit 1
fi

# Ensure that git tag exists for the current commit
if [[ -z $(git tag --points-at HEAD) ]]; then
  echo "Tag the current commit first"
  exit 1
fi

# Parse the tag from main.go
# shellcheck disable=SC2002
CUR_TAG=$(cat main.go | grep 'Version = ' | cut -d '"' -f 2)

# Ensure that this matches the current commit's tag in git
if [[ $CUR_TAG != $(git tag --points-at HEAD) ]]; then
  echo "The main.go version does not match the current commit's tag"
  exit 1
fi

# Check that the tag follows semver
if [[ ! $CUR_TAG =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Tag does not follow semver (e.g. v1.2.3)"
  exit 1
fi

# Bump the last number in the semver tag
# shellcheck disable=SC2002
NEXT_TAG=$(echo "$CUR_TAG" | awk -F. '{$NF = $NF + 1;} 1' | sed 's/ /./g')

# Check that the tag does not already exist on origin
if [[ $(git ls-remote --tags origin "NEXT_TAG") ]]; then
  echo "New tag already exists on origin"
  exit 1
fi

# Update the tag in main.go
sed -i "s/$CUR_TAG/$NEXT_TAG/g" main.go

# calculate the tag that the previous version should have
# shellcheck disable=SC2002
PREV_TAG=$(echo "$CUR_TAG" | awk -F. '{$NF = $NF - 1;} 1' | sed 's/ /./g')

# Update the tag in the README
sed -i "s/$PREV_TAG/$CUR_TAG/g" README.md

