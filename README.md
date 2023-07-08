# FlyCD

## Goals

FlyCD is a tool designed to add ArgoCD/Flux style git-ops support for Fly.io. Although its fully automated git-ops
functionality is still to be implemented, the following are the features it aims to provide:

* Extending the regular fly.io fly.toml specifications with additional configuration parameters,
  removing the need for running flyctl commands manually.

* FlyCD is installed and runs as any other fly.io app inside the fly.io environment you install it in, listening to
  webhooks from git pushes,
  and grabbing the latest versions (or specific versions) of your applications from git, and deploying them to fly.io.

* FlyCD spec format is a superset of regular fly.io toml files (although fly.io uses toml and flycd uses yaml, yet are
  1:1 convertible between, and who knows, flycd might use toml in the future if flycd author(s) stop hating toml :)

## Current state

FlyCD is built on the principle of bootstrapping itself.

* It can install/launch/deploy specifications to an existing fly.io environment
* It can operate as a manual Git-Ops CLI tool for deploying fly.io apps with a superset of fly.toml, such as:
    * specifying a source git repo (+optional branch/tag/commit) to deploy the app from
    * the target organisation to deploy to
* It can deploy many apps at the same time. Simply point it to a directory structure/hierarchy containing multiple
  app.yaml files, and flycd will traverse the structure recursively and deploy each app
* It can currently install itself into an existing fly.io environment (although it doesn't do anything yet once
  installed :D)
* It can understand and write fly.toml files from flycd spec (fly)

## Current issues

* Lots of implementation is still missing!
* SUPER HACKY code right now, just a 1 day hack so far with most work delegated to shell commands instead of proper go
  libraries :D
