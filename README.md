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

1. Run `go install github.com/gigurra/flycd@<version>` (currently `v0.0.46`)
2. Run `flycd deploy <fs path>` to deploy a configuration (single app or structure with many projects and apps, you
   decide)
3. (Optional) Installing flycd as an app in your fly.io account or as a daemon somewhere else where you prefer to have
   it running.
    * Method 1: Run `flycd install --project-path <fs path>` to install flycd into your fly.io environment.
      This will create a new fly.io app running flycd in monitoring mode/webhook listening mode. The `install` command
      will automatically issue a fly.io API token for itself, and store it as an app secret in fly.io. You can ssh into
      your flycd container and copy it from there if you want to use it for other purposes (you prob shouldn't) or just
      locally verify that it works.
        * To make it able to clone private git repos, create a fly.io secret called `FLY_SSH_PRIVATE_KEY`
        * `flycd install` is also how you upgrade a fly.io deployed flycd installation to a new flycd version.
    * Method 2: Use the flycd docker image (https://hub.docker.com/r/gigurra/flycd) and mount a project.yaml into
      the container's `/flycd/projects/project.yaml`, where your `project.yaml` points to your config repo. You could
      also customize the image to your liking (see [Dockerfile](Dockerfile)).
    * There are many other ways you could do this as well
4. Optional: Add a webhook to your GitHub repo(s), pointing to your flycd app's url,
   e.g. the default POST path `https://<your-flycd-app-name>.fly.dev/webhook`, which currently just supports GitHub push
   webhooks.

### Using the flycd CLI

The best is probably to check the `--help` output:

```
$flycd --help

Starting FlyCD v0.0.46...
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

#### File system layout

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

#### project.yaml

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

#### app.yaml

Further down the tree we have app directories with `app.yaml` files (or more `project.yaml` files if you want to have
recursive projects/projects-in-projects :P).

Tip: Easy ways to create your own app.yaml files:

* Copy and modify an example
* Or use the `flycd convert` command to convert your existing `fly.toml` files to `app.yaml` files.
* Or download existing definitions `fly config show [-a <your-app-name>] | yq -P > app.yaml`

These might look something like this:

```yaml
# app.yaml containing the regular fly.io app config + FlyCD's additional fields
# NOTE: Most of the below is optional! (essentially, fly.io dictates which fields are optional, and FlyCD will try not to enforce too much)
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
  ram_mb: 256 # override the setting that the app was created with. Only works for process 'app'
  cpu_cores: 1 # override the setting that the app was created with. Only works for process 'app'
  cpu_type: "shared" # override the setting that the app was created with. Only works for process 'app'
  
## Optional env vars
env:
  PORT: "8081"

## Optional volumes and mounts. It has some limitations:
# - Currently, fly.io only supports 1 mount point and volume per machine.
#   In principle, FlyCD supports arbitrary numbers of both, should fly.io change this in the future.
# - The number of volume instances will be created to match the number of app instances (machines). 
#   This is the max between current existing instances/machines and the min_machines_running field,
#   but no less than 1. So if you have min_machines_running: 0, you will always have at least 1 volume instance.
volumes:
  - name: my-volume
    # size_gb: can be increased later, but cannot be reduced (fly.io limitation)
    size_gb: 10
    # count: should be enough to cover the number of app instances.
    # FlyCD will automatically use actual app instance count if it's higher than this.
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
  # optional, if you want to use a pre-built image instead of building from source.
  # fly.io supports this only for public docker registries (e.g. dockerhub)
  # probably best used in combination with source type: local
  #image: some/docker-image:tag 

# Optional config for secrets. Here you define what the secrets should be created in the fly.io app
# and where to get the value from. Currently only supports getting the value from env vars on the host 
# running FlyCD itself, or (for test purposes) as raw in-config/inline plaintext.
# It works by creating a fly.io secret with the same name as the secret config entry.
secrets:
  # Secrets forwarded from env vars on the host running FlyCD (e.g. your local machine or installed FlyCD instance) 
  - name: SOME_SECRET_FWD # this is the name/key the secret will have in the fly.io app
    type: env # this tells FlyCD how to extract the secret value
    env: SOME_SECRET # optional name of the source env var on the host running FlyCD. Defaults to the name of the secret
  - name: SOME_SECRET_2
    type: env
  # unencrypted plaintext in config. NOT recommended. For testing only!
  - name: SOME_TEST_SECRET
    type: raw
    raw: secret

# Optional networking config
network:
  auto_prune_ips: true # deletes all ips for this app not listed below
  ips:
    - v: v6
      Private: true
      #Network: ...
      #Region: ... #default = global
    - v: v4
      Shared: true

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

#### Deploying the example

* Test deploy your config with `flycd deploy .`
* Before installing it into your fly.io account with `flycd install --project-path .`

FlyCD will convert the `app.yaml` back to `fly.toml` before deploying to fly.io, and will keep all fields you put in
it (i.e. flycd doesn't have to implement the full fly.io domain model). There are several reasons flycd doesn't just use
a `fly.toml` instead of `app.yaml`. One reason is because `flycd` uses the fly.io cli (`fly`/`flyctl`) under the hood,
and the fly.io cli actually modifies the `fly.toml` in place when deploying :S. Another is that we want to re-use data
within the config, and `.toml` is not a very good format for that. There are probably more reasons, some
subjective, like the author of FlyCD just likes yaml more than toml :D.

#### More examples

Check the [examples](examples) directory for more ideas. You can also check the [tests](test) directory for some special
cases

### Webhooks

There are two kinds of webhooks:

* Configuration repo webhooks
* App repo webhooks

FlyCD listens to both on the same path (`/webhook` by default), but they are handled differently.

Currently only GitHub push webhooks are supported - simply click settings on your github repo's page and add a webhook
to your flycd installation's url (e.g. `https://<your-flycd-app-name>.fly.dev/webhook`).

#### Configuration repo webhooks

Configuration repos are where you store the flycd configuration files (app.yaml, project.yaml, etc).
If your cloud setup is fairly small/static, or you want to manage its configuration without flycd, you can skip setting
up webhooks from this repo. In that case you need to ensure you re-run `flycd install` every time your configuration
changes. Having a configuration repo with webhooks, means less manual interation with flycd and less CD configuration,
but the choice is yours.

When a configuration repo (=project repo) webhook is triggered, flycd traverse the entire tree structure of projects and
apps enclosed by the repo/project, and evaluate all apps inside if the config change has made it necessary to re-deploy
them.

#### App repo webhooks

App repos are where you store your app code, and optionally part of your app configuration (but never app.yaml).
To have flycd auto deploy your apps, you need to set up webhooks from your app repos to flycd. Just enable GitHub's
regular push webhook functionality and point it to your flycd app's webhook url.

FlyCD will then automatically fetch the latest version (or specific version, if set in your config repo) of your apps
and deploy them and any updates to their configuration.

When an app repo webhook is triggered, flycd will will only evaluate that specific app for changes to know if it needs
to be re-deployed.

### Pruning policies

Currently, FlyCD never deletes any resources from your fly.io account. FlyCD just adds and updates existing resources.
This means for example that if you delete an app from your config, it will still exist in your fly.io account. The same
goes for environment variables, secrets, etc.

This may change in the future, and if so, FlyCD will have this an opt-in feature (just like auto pruning with ArgoCD).

### Storage and State

FlyCD is currently mostly stateless - it doesn't keep any persistent storage. Instead it uses fly.io app environment
variables to store configuration and app repo hashes to later determine if a re-deploy is required or not. You can
override this using `flycd deploy <path> --force`.

For performance and consistency reasons flycd will probably become stateful at some point in the future.

NOTE: You should never run more than 1 flycd instance. This is because flycd currently is quite basic in determining
what changes could conflict or cause race conditions with each other if deployed in parallel. FlyCD just queues all
changes/webhooks to a single im-mem worker, so they are executed in order. This is not ideal, but it works for now.

## Where it probably needs some improvement

* Performance: It needs some way of determining if webhooks interfere with each other. Right now they are just executed
  one at a time (they are queued to a single worker to avoid races)..
* Performance: It needs some way of determining what parts of the config tree have changed, and only traverse and
  evaluate those parts. Right now it traverses the whole config tree every time when receiving a webhook and looks for
  potential modifications. (it doesn't deploy everything, but it need to traverse the whole tree)
* Consistency: It needs some persistence of incoming webhooks. Right now if FlyCD goes down during a deployment, the
  deployment will be lost.
* Consistency: It needs regular jobs/auto sync for apps that don't send webhooks, like's ArgoCD's 3-minute polling.
* Consistency: Support for pruning policies.
* Security: It needs some security validation of webhooks from GitHub :D. Currently, there is none so DOS attacks are
  trivial to create :S.
* Security: Better/more secrets providers
* Non-Github: It currently only supports webhooks from git repos at GitHub.
* Authentication: It currently only supports authentication via git over ssh, and only a single private key can be
  loaded.
    * It would be nice to support other authentication methods, such tokens for https.
* Quality: Could probably do with some cleanup and refactoring, and maybe some more automated tests

### Some immediate TODOs

* Support per-region machine size configurations (ram, cpu)
* Support multiprocess apps (flycd currently only supports 'app' for figuring out when scale up the number of machines
  and volumes)
* More practical ways to configure Machine types, ram & cpu modifications
    * Right now it is possible, but only by setting the `launch_params` and/or `deploy_params` fields (see examples)
* Cron style jobs by leveraging fly.io's scheduled machines
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
go build .
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