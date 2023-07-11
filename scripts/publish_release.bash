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

# Check that no git tag exists for the current commit
if [[ $(git tag --points-at HEAD) ]]; then
  echo "Tag for the current commit already exists"
  exit 1
fi

# Parse the tag from main.go
# shellcheck disable=SC2002
TAG=$(cat main.go | grep 'Version = ' | cut -d '"' -f 2)

# Check that the tag follows semver
if [[ ! $TAG =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Tag does not follow semver (e.g. v1.2.3)"
  exit 1
fi

# Check that the tag does not already exist on origin
if [[ $(git ls-remote --tags origin "$TAG") ]]; then
  echo "Tag already exists on origin"
  exit 1
fi

echo "Tag is available, building image first to make sure it works"

# Image name and tag
IMAGE_NAME=gigurra/flycd:$TAG

# Specify platform
PLATFORM=linux/amd64

# Build the image
IMAGE_NAME=gigurra/flycd:$TAG

# Build the Docker image
docker build --platform $PLATFORM -t "$IMAGE_NAME" .

# Tag the commit
git tag "$TAG"

# Push the commit and tag to origin
git push --tags origin master

# Push the image to docker hub
docker push "$IMAGE_NAME"



