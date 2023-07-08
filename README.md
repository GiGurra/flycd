# FlyCD

## Goals

FlyCD is a tool designed to add ArgoCD/Flux style git-ops support for Fly.io. Although its fully automated git-ops
functionality is still to be implemented, the following are the features it aims to provide:

* Extending the regular fly.io fly.toml specifications with additional configuration parameters,
  removing the need for running **_any_** flyctl commands manually.

* FlyCD is installed and runs as any other fly.io app inside the fly.io environment you install it in, listening to
  webhooks from git pushes,
  and grabbing the latest versions (or specific versions) of your applications from git, and deploying them to fly.io.

* FlyCD spec format is a strict superset of regular fly.io toml files. Although fly.io uses toml and flycd uses yaml,
  they are 1:1 convertible between, and who knows, flycd might use toml in the future if flycd author(s) stop hating
  toml :).

```
$flycd --help

Starting FlyCD latest...
Complete documentation is available at https://github.com/gigurra/flycd

Usage:
  flycd [flags]
  flycd [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  deploy      Manually deploy a single flycd app, or all flycd apps inside a folder
  help        Help about any command
  install     Install flycd into your fly.io account, listening to webhooks from this cfg repo and your app repos
  monitor     (Used when installed in fly.io env) Monitors flycd apps, listens to webhooks, grabs new states from git, etc
  uninstall   Uninstall flycd from your fly.io account
  upgrade     Upgrade your flycd installation in your fly.io account to the latest version

Flags:
  -h, --help   help for flycd

Use "flycd [command] --help" for more information about a command.
FlyCD latest exiting normally, bye!
```

## Current state

FlyCD is built on the principle of bootstrapping itself.

* It can install/launch/deploy specifications to an existing fly.io environment
* It can operate as a manual Git-Ops CLI tool for deploying fly.io apps with a superset of fly.toml, such as:
    * specifying a source git repo (+optional branch/tag/commit) to deploy the app from
    * the target organisation to deploy to
* It can deploy many apps at the same time. Simply point it to a directory structure/hierarchy containing multiple
  app.yaml files, and flycd will traverse the structure recursively, clone each app's source repo and deploy each app
* It can currently install itself into an existing fly.io environment (although it doesn't do anything yet once
  installed :D)

## Current issues

* Lots of implementation is still missing!
* SUPER HACKY code right now, just a 1 day hack so far with most work delegated to shell commands instead of proper go
  libraries :D
  * Lots of refactoring needed!

## Current incomplete TODO list

* All of the above ☝️☝️☝️
* ssh keys/credentials for cloning app repos
* listening to webhooks, and figuring out which app/apps it relates to
* better error handling :S
* better logging
* some status views or status APIs maybe
  * if someone ever has time to build a UI :D
* Volumes & mounts
* Secrets
* Machine types, ram & cpu modifications
* fly.io native postgres, redis, etc...
