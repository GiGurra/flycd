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
    * figures out if a deployment is actually warranted, by comparing `sha1sum` of app.yaml + git hash of app repo with hashes
      saved to fly.io env for the app (using app env vars for this)
* It can deploy many apps at the same time. Simply point it to a directory structure/hierarchy containing multiple
  app.yaml files, and flycd will traverse the structure recursively, clone each app's source repo and deploy each app
* It can currently install itself into an existing fly.io environment (although it doesn't do anything yet once
  installed :D)

**I have no idea if I will have time or interest in continuing this project until it reaches a useful state :D.**
Consider it proof of concept, and nothing more. I have spent about 1 day on it so far.

## Current issues

* Lots of implementation is still missing!
* SUPER HACKY code right now, just a one-day hack so far with most work delegated to shell commands instead of proper go
  libraries :D
    * Lots of refactoring needed!
* This functionality might already exist/I might be reinventing the wheel here - we will see what is written in the
  discussion thread over at fly.io community forums. 
  * see https://community.fly.io/t/simple-self-contained-argocd-style-git-ops-with-fly-io-what-are-the-options-poc-flycd-app/14032

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

## Using it

### Setting up your own config repo

1. The two ways I'd recommend, either:
    * Fork this repo and run `go install .`
    * `go install github.com/gigurra/flycd@v0.0.8`
2. Modify the contents of the `project` folder (or create one), and add the app specifications you like
3. Run `flycd deploy projects`

### Installing flycd git-ops app to your fly.io env (not yet implemented)

Not yet implemented.

The following just installs a blank nginx server and creates an org token (prints it to your terminal).

```
flycd install tempflycd personal arn
```

```
fly apps list

NAME            OWNER           STATUS          PLATFORM        LATEST DEPLOY        
tempflycd       personal        deployed        machines        30m3s ago  
```

## Sample project structure

```
projects/
└── example-project1
    ├── local_app
    │   ├── Dockerfile
    │   └── app.yaml
    └── x_git_app
        └── app.yaml
```
## Sample flycd app.yaml

```yaml
############################
## Basic configuration
app: &app example-project1-git-app-foobar12341

# Point to a repo. You are expected to mount the necessary ssh keys to the container
source:
  repo: "https://github.com/GiGurra/flycd-nginx-test"
  #path: "path/to/app" # optional
  ref:
    branch: "main"
    #tag: "v1.2.3"
    #commit: "fae2e1f1f578ddda681f09137dbae831bde84fe7"
  type: "git"

############################
## Optional configuration
env:
  ENV: "development"

primary_region: "arn" # default region for tests is Stockholm/Sweden
services:
  - internal_port: 80
    protocol: "tcp"
    force_https: true
    auto_stop_machines: true
    auto_start_machines: true
    min_machines_running: 1
    concurrency:
      type: "requests"
      soft_limit: 200
      hard_limit: 250
    ports:
      - handlers: [ "http" ]
        port: 80
        force_https: true
      - handlers: [ "tls", "http" ]
        port: 443

# Modify to your needs. By default, we will create a new fly.io
# app without any user interaction/confirmation.
# For the most simple apps, you probably don't need to modify these at all
launch_params:
  - "--ha=false"
  - "--auto-confirm"
  - "--now"
  - "--copy-config"
  - "--name"
  - *app

# Modify to your needs. By default, we will deploy the fly.io
# app without any user interaction/confirmation.
# For the most simple apps, you probably don't need to modify these at all
deploy_params: [ ]

```