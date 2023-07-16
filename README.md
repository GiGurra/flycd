# FlyCD

## Overview

FlyCD adds ArgoCD/Flux style git-ops support for Fly.io:

* Extending the [fly.io configuration file structure](https://fly.io/docs/reference/configuration/) to eliminate the
  need for manual execution of fly.io CLI commands.

* FlyCD operates like any other fly.io app within the fly.io environment in which it's installed. It listens to webhooks
  from git pushes, fetches the most recent (or specific) versions of your apps from git, and deploys them to fly.io.

* Keeping app repos separate from your environment configuration repos (or put everything in the same repo... if you
  want to :D)

The illustration below gives an idea of FlyCD enabled configuration:

![alt text](https://raw.githubusercontent.com/GiGurra/flycd/master/concept.svg)

## How to use it

### Installation

1. Run `go install github.com/gigurra/flycd@<version>` (currently `v0.0.36`)
2. Run `flycd deploy <fs path>` to deploy a configuration (single app or folder structure, you decide)
3. Optional: Run `flycd install --project-path <fs path>` to install flycd into your fly.io environment.
   This will create a new fly.io app running flycd in monitoring mode/webhook listening mode. The `install` command will
   automatically issue a fly.io API token for itself, and store it as an app secret in fly.io. You can ssh into your
   flycd container and copy it from there if you want to use it for other purposes (you prob shouldn't) or just locally
   verify that it works.
    * To make it able to clone private git repos, create a fly.io secret called `FLY_SSH_PRIVATE_KEY`
    * `flycd install` is also how you upgrade a fly.io deployed flycd installation to a new flycd version.
4. Optional: Add a webhook to your GitHub repo(s), pointing to your flycd app's url,
   e.g. the default POST path `https://<your-flycd-app-name>.fly.dev/webhook`, which currently just supports GitHub push
   webhooks.

### Using the flycd CLI

The best is probably to check the `--help` output:

```
$flycd --help

Starting FlyCD v0.0.37...
Complete documentation is available at https://github.com/gigurra/flycd

Usage:
  flycd [flags]
  flycd [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  convert     Convert app/apps from fly.toml(s) to app.yaml(s)
  deploy      Manually deploy a single flycd app, or all flycd apps inside a folder
  help        Help about any command
  install     Install FlyCD into your fly.io account, listening to webhooks from this cfg repo and your app repos
  monitor     (Used when installed in fly.io env) Monitors flycd apps, listens to webhooks, grabs new states from git, etc
  repos       Traverse the project structure and list all git repos referenced. Useful for finding your dependencies (and setting up webhooks).

Flags:
  -h, --help   help for flycd

Use "flycd [command] --help" for more information about a command.
```

### Configuration examples

Suppose you have a GitHub repo called `my-cloud` in `my-org` (https://github.com/my-org/my-cloud), looking something
like this

```
.
├── cloud-x
│   └── stage
│   |   ├── some-backend
│   |   │   └── app.yaml
│   |   ├── some-frontend
│   |   │   └── app.yaml
│   |   └── some-db
│   |       ├── app.yaml
│   |       ├── Dockerfile
│   |       ├── settings.json
│   |       └── start-script.sh
│   └── prod
│       ├── some-backend
│       │   └── app.yaml
│       ├── some-frontend
│       │   └── app.yaml
│       └── some-db
│           ├── app.yaml
│           ├── Dockerfile
│           ├── settings.json
│           └── start-script.sh
├── cloud-y
│   ├── auth-proxy
│   │   ├── app.yaml
│   │   ├── Dockerfile
│   │   ├── nginx.conf
│   │   └── start_proxy.bash
│   └── some-backend
│       ├── app.yaml
│       └── Dockerfile
└── project.yaml
```

The project structure doesn't have to look like this (flycd walks it recursively), so the above is just an example. Here
we have defined a top level of two "clouds" (cloud-x and cloud-y), cloud-x also having two environments (stage and
prod).

At the top we have a `project.yaml` file which lets FlyCD know where this is located. We could also have
local `project.yaml` files further down the tree to set common parameters/values for configurations for that subtree.
You can also skip having `project.yaml` files entirely, though in that case you need to only use manual deployments, or
manually redeploy your installed flycd instance to your fly.io environment for every config change.

```yaml
# project.yaml
project: "my-org-cloud"
source:
  type: git
  repo: "git@github.com:my-org/my-cloud"

# Optional common parameters that affect all apps within this project
# This also works for nested projects (projects within projects)
common:
  app_defaults:
    http_service:
      min_machines_running: 2
  app_overrides:
    org: my-org-2
    primary_region: ams
  substitutions: # raw string replacements in all app.yaml files
    someRegex: "strReplacement" # could prob be improved
```

Further down the tree we have app directories with `app.yaml` files (or more `project.yaml` files if you want to have
recursive projects/projects-in-projects :P).

Tip: Easy ways to create your own app.yaml files:

* Copy and modify an example
* Or use the `flycd convert` command to convert your existing `fly.toml` files to `app.yaml` files.
* Or download existing definitions `fly config show [-a <your-app-name>] | yq -P > app.yaml`

These might look something like this:

```yaml
# app.yaml containing the regular fly.io app config + flycd's additional fields
# NOTE: Most of the below is optional! (essentially, fly.io dictates which fields are optional, and flycd will try not to enforce too much)
app: &app cloud-x--prod--some-backend # Unique dns name at <app>.fly.dev, as is the case with fly.io apps with automatic dns

# All regular fly.io config file fields are supported (by preserving untyped config tree in parallel with typed).
# This is just an example with fly.io's 'http_service'. You can also use 'services'.
http_service:
  auto_start_machines: true
  auto_stop_machines: true
  force_https: true
  internal_port: 8081
  min_machines_running: 0
  processes:
    - app
primary_region: &primary_region arn

## source: The most important field for FlyCD. It tells FlyCD where to find the app's git repo.
# You can also set it to type "local" and point it to a local directory within the config repo/project.
# see examples on how to configure local vs git types, and the use of their path and ref parameters
source:
  type: git # or local
  #path: "some/path/within/local/or/repo"
  repo: "git@github.com:my-org/my-app" # only needed for type git
  #ref: # only applies for type git
  #  commit: "some-commit-hash"
  #  branch: "some-branch-name"
  #  tag: "some-tag-name"

######################################
## more optional example config below

# extra regions besides the primary region where this app will run
# note: All volumes and mounts will be created in all regions (primary + extra regions).  
extra_regions:
  - ams

# optional machine config. Currently only supports count
machines:
  count: 2 # default count for all regions
  count_per_region:
    ams: 3 # override count for a specific region

## Optional env vars
env:
  PORT: "8081"

## Optional volumes and mounts. It has some limitations:
# - Currently, fly.io only supports 1 mount point and volume per machine.
#   In principle, flycd supports arbitrary numbers of both, should fly.io change this in the future.
# - The number of volume instances will be created to match the number of app instances (machines). 
#   This is the max between current existing instances/machines and the min_machines_running field,
#   but no less than 1. So if you have min_machines_running: 0, you will always have at least 1 volume instance.
volumes:
  - name: my-volume
    # size_gb: can be increased later, but cannot be reduced (fly.io limitation)
    size_gb: 10
    # count: should be enough to cover the number of app instances.
    # flycd will automatically use actual app instance count if it's higher than this.
    # However, if fly.io wants to scale higher than this, it won't be able to until you increase the count.
    # For most applications you probably won't use fly.io auto-scaling in combination with volumes :S 
    count: 3
mounts:
  - destination: /mnt/my-volume-goes-here
    source: my-volume

## Optional build config (this is something fly.io cli can generate for you)
build:
  builder: paketobuildpacks/builder:base
  buildpacks:
    - gcr.io/paketo-buildpacks/go

vm_size: &vm_size "shared-cpu-1x"

org: &org personal

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
  - "--region"
  - *primary_region
  - "--org"
  - *org
  - "--vm-size"
  - *vm_size

# Modify to your needs. By default, we will deploy the fly.io
# app without any user interaction/confirmation.
# For the most simple apps, you probably don't need to modify these at all
deploy_params:
  - "--ha=false"
  - "--vm-size"
  - *vm_size
```

* Test deploy your config with `flycd deploy .`
* Before installing it into your fly.io account with `flycd install --project-path .`

FlyCD will convert the `app.yaml` back to `fly.toml` before deploying to fly.io, and will keep all fields you put in
it (i.e. flycd doesn't have to implement the full fly.io domain model). There are several reasons flycd doesn't just use
a `fly.toml` instead of `app.yaml`. One reason is because `flycd` uses the fly.io cli (`fly`/`flyctl`) under the hood,
and the fly.io cli actually modifies the `fly.toml` in place when deploying :S. Another is that we want to re-use data
within the config, and `.toml` is not a very good format for that. There are probably more reasons, some
subjective, like the author of FlyCD just likes yaml more than toml :D.

Check the [examples](examples) directory for more ideas.

## Where it needs improvement

* Performance: It needs some way of determining if webhooks interfere with each other. Right now they are just executed
  one at a time (they are queued to a single worker to avoid races)..
* Performance: It needs some way of determining what parts of the config tree have changed, and only traverse and
  evaluate those parts. Right now it traverses the whole config tree every time when receiving a webhook and looks for
  potential modifications. (it doesn't deploy everything, but it might need to traverse potentially the whole tree)
* Consistency: It needs some persistence of incoming webhooks. Right now if FlyCD goes down during a deployment, the
  deployment will be lost.
* Consistency: It needs regular jobs/auto sync for apps that don't send webhooks, like's ArgoCD's 3-minute polling.
* Security: It needs some security validation of webhooks from GitHub :D. Currently, there is none so DOS attacks are
  trivial to create :S.
* Non-Github: It currently only supports webhooks from git repos at GitHub.
* Non-Git sources: It might be useful to also support regular docker images and different docker registries (right now
  to deploy from an image, you have to create a proxy Dockerfile, Or create a single line inline Dockerfile in your
  app.yaml, which is a bit ugly).
    * Currently, there is no support for private image registries. You have to point to a private git repo instead
      containing a Dockerfile.
* Authentication: It currently only supports authentication via git over ssh, and only a single private key can be
  loaded.
    * It would be nice to support other authentication methods, such tokens for https.
* Quality: Could probably do with some cleanup and refactoring, and maybe some more automated tests

### Some immediate TODOs

* Support for creation/updating fly.io secrets (not sure how though :S)
* Support per region machine size configurations (ram, cpu)
* Support multi-process apps (flycd currently only supports 'app' for machine and volume scaling)
* More practical ways to configure Machine types, ram & cpu modifications
      * Right now it is possible, but only by setting the `launch_params` and/or `deploy_params` fields (see examples)
* better error handling :S
* better logging
* fly.io native postgres, redis, etc...

## Building from source

#### Prerequisites

* Go 1.20
* [Mockery](https://github.com/vektra/mockery) (if running tests, for generating mocks)
* Docker (if building the standalone docker image)

#### To build the application:

```
go build ./...
```

#### To install `flycd` from source:

```
go install .
```

#### To run the application from source without installing:

```
go run .
```

#### To run tests

```
mockery # generates mocks
go test ./...
```

#### To build the standalone docker image:

```
docker build -t yourName/flycd:latest .
```

## Links/References

* [Git-Ops](https://www.redhat.com/en/topics/devops/what-is-gitops#:~:text=GitOps%20uses%20Git%20repositories%20as,set%20for%20the%20application%20framework.)
* [Argo-CD](https://argoproj.github.io/cd/)
* [GitHub webhooks](https://docs.github.com/en/webhooks-and-events/webhooks/about-webhooks)

## License

MIT

## Contributing

Contact [@gigurra](https://github.com/GiGurra) if you want to contribute, or just create a PR.