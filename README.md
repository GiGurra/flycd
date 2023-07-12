# FlyCD

## Overview

FlyCD adds ArgoCD/Flux style git-ops support for Fly.io, through the following:

* Extending the standard fly.io fly.toml specifications with supplementary configuration parameters to eliminate the
  need
  for manual execution of any fly.io CLI commands.

* Separatting the processes of application development and environment deployment/composition.
    * You can develop the code in one repository and push updates to your app, keeping the repository devoid of any
      environment-specific configuration.
    * It allows you to maintain numerous fly.io environments that utilize the app in varying versions and
      configurations, eliminating the necessity of embedding environment-specific configurations into your app.
    * It offers the flexibility to deploy or reference any app (or version of any app), whether from your repositories
      or owned by others, and separately compose the cloud environment from the application development.

* FlyCD operates like any other fly.io app within the fly.io environment in which it's installed. It listens to webhooks
  from git pushes, fetches the most recent (or particular) versions of your apps from git, and deploys them to fly.io.

* The specification format of FlyCD is a strict superset of regular fly.io toml files. Despite fly.io using toml and
  FlyCD utilizing yaml, both formats are interchangeable. In the future, FlyCD may adopt toml if the authors reconsider
  their stance on toml.

The illustration below gives an idea of FlyCD enabled configuration:

![alt text](https://raw.githubusercontent.com/GiGurra/flycd/master/concept.svg)

```
$flycd --help

Starting FlyCD v0.0.18...
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

Flags:
  -h, --help   help for flycd

Use "flycd [command] --help" for more information about a command.
FlyCD v0.0.18 exiting normally, bye!
```

## Current state

FlyCD is built on the principle of bootstrapping itself.

* It can install itself to an existing fly.io environment and point to a config repo
    * where it listens and acts on both config repo and app repo webhooks
* It can also operate as a manual CLI tool for deploying fly.io apps with a superset of fly.toml, such as:
    * specifying a source git repo (+optional branch/tag/commit) to deploy the app from
    * the target organisation to deploy to
    * figures out if a deployment is actually warranted, by configuration folder and app folder checksum/app git hash
      with hashes saved to fly.io env for the app (using app env vars for this)
* It can deploy many apps at the same time. Simply point it to a directory structure/hierarchy containing multiple
  app.yaml files, and flycd will traverse the structure recursively, clone each app's source repo and deploy each app

### Where it needs improvement

* It needs some persistence and queueing of incoming webhook commands to no run into data races, or lose data if flycd
  crashes or is re-deployed :S. Currently, it just spins up a new go-routine for each webhook request.
* It needs a better way to scan existing available apps than re-read all of their specs from disk on every webhook
  event.
* It needs regular jobs/auto sync for apps that don't send webhooks, like 3rd party tools where we probably can't add
  webhooks.
* It needs some security validation of webhooks from GitHub :D. Currently, there is none so DOS attacks are trivial to
  create :S.
* It currently only supports git repos and file system directories as app sources. It might be useful to also support
  regular docker images (right now to deploy from an image, you have to create a proxy Dockerfile)

**I have no idea if I will have time or interest in continuing this project until it reaches a useful state :D.**
Consider it proof of concept, and nothing more.

## Other concerns

* Pretty hacky code right now, without too many automated tests, just a 1-week hack so far with a lot of work delegated
  to shell commands instead of proper go libraries :D
    * Refactoring needed!
* This functionality might already exist/I might be reinventing the wheel here - we will see what is written in the
  discussion thread over at fly.io community forums.
    * https://community.fly.io/t/simple-self-contained-argocd-style-git-ops-with-fly-io-what-are-the-options-poc-flycd-app/14032
* Only supports GitHub webhooks
* The current handling of private repos is hacky (currently you have to manually save a fly.io secret with a private key
  for git over ssh)

## Current incomplete TODO list

* Fix current issues above ☝️☝️☝️
* better error handling :S
* better logging
* some status views or status APIs maybe
    * if someone ever has time to build a UI :D
* Volumes & mounts
* Secrets
* Machine types, ram & cpu modifications
* fly.io native postgres, redis, etc...

## Getting started

### Quick setup

1. Run `go install github.com/gigurra/flycd@<version>` (currently `v0.0.18`)
2. Run `flycd deploy <your projects folder>` to ensure it deploys things the way you expect
3. Run `flycd install --project-path <your projects folder>` to install flycd into your fly.io environment.
   This will create a new fly.io app running flycd in monitoring mode/webhook listening mode. The `install` command will
   automatically issue a fly.io API token for itself, and store it as an app secret in fly.io. You can ssh into your
   flycd container and copy it from there if you want to use it for other purposes (you prob shouldn't) or just locally
   verify that it works.
4. Add a webhook to your git repo, pointing to your flycd app's url,
   e.g. the default POST path `https://<your-flycd-app-name>.fly.dev/webhook`, which currently just supports GitHub push
   webhooks.
5. Watch the magic happen!

### Sample project structure

Check the [examples](examples) directory for some ideas.