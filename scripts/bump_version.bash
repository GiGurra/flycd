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

# Parse the tag from main.go
# shellcheck disable=SC2002
CUR_TAG=$(cat main.go | grep 'Version = ' | cut -d '"' -f 2)

# Check that the tag follows semver
if [[ ! $CUR_TAG =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Tag does not follow semver (e.g. v1.2.3)"
  exit 1
fi

# Bump the last number in the semver tag
# shellcheck disable=SC2002
NEXT_TAG=$(echo "$CUR_TAG" | awk -F. '{$NF = $NF + 1;} 1' | sed 's/ /./g')

# calculate the tag that the previous version should have
# shellcheck disable=SC2002
PREV_TAG=$(echo "$CUR_TAG" | awk -F. '{$NF = $NF - 1;} 1' | sed 's/ /./g')

# Ensure that the current tag exists in git, it could be any previous commit that has this tag
if git rev-parse "$CUR_TAG" >/dev/null 2>&1
then
  echo "PREV_TAG: $PREV_TAG"
  echo "CUR_TAG: $CUR_TAG"
  echo "NEXT_TAG: $NEXT_TAG"
else
  echo "There does not seem to be any commit with the current tag(main.go.Version: $CUR_TAG). Can't advance"
  exit 1
fi

# Check that the tag does not already exist on origin
if [[ $(git ls-remote --tags origin "NEXT_TAG") ]]; then
  echo "$NEXT_TAG tag already exists on origin"
  exit 1
fi

# Update the tag in main.go
sed -i "s/$CUR_TAG/$NEXT_TAG/g" main.go

# Update the tag in the README
sed -i "s/$PREV_TAG/$CUR_TAG/g" README.md

# add the changes to git
git add main.go README.md

# commit the changes
git commit -m "Bump version from $CUR_TAG to $NEXT_TAG"

git push

