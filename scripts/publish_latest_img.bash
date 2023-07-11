#!/usr/bin/env bash

set -e

# check that the build works
go clean ./...
go build ./...
go test ./...

TAG=latest

# Image name and tag
IMAGE_NAME=gigurra/flycd:$TAG

# Specify platform
PLATFORM=linux/amd64

# Build the image
IMAGE_NAME=gigurra/flycd:$TAG

# Build the Docker image
docker build --platform $PLATFORM -t "$IMAGE_NAME" .

## Tag the commit
#git tag "$TAG"

## Push the commit and tag to origin
#git push --tags origin master

# Push the image to docker hub
docker push "$IMAGE_NAME"



