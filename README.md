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

## So how do I use it?

The best is probably to check the `--help` output:

```
$flycd --help

Starting FlyCD v0.0.30...
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

## Installation

1. Run `go install github.com/gigurra/flycd@<version>` (currently `v0.0.30`)
2. Run `flycd deploy <fs path>` to deploy a configuration (single app or folder structure, you decide)
3. Optional: Run `flycd install --project-path <fs path>` to install flycd into your fly.io environment.
   This will create a new fly.io app running flycd in monitoring mode/webhook listening mode. The `install` command will
   automatically issue a fly.io API token for itself, and store it as an app secret in fly.io. You can ssh into your
   flycd container and copy it from there if you want to use it for other purposes (you prob shouldn't) or just locally
   verify that it works.
    * To make it able to clone private git repos, create a fly.io secret called `FLY_SSH_PRIVATE_KEY`
4. Optional: Add a webhook to your GitHub repo(s), pointing to your flycd app's url,
   e.g. the default POST path `https://<your-flycd-app-name>.fly.dev/webhook`, which currently just supports GitHub push
   webhooks.

## Configuration examples

Check the [examples](examples) directory for some ideas.

## Where it needs improvement

* Performance: It needs some way of determining if webhooks interfere with each other. Right now they are just executed
  one at a time (they are queued to a single worker to avoid races)..
* Consistency: It needs some persistence of incoming webhooks. Right now if FlyCD goes down during a deployment, the
  deployment will be lost.
* Consistency: It needs regular jobs/auto sync for apps that don't send webhooks, like 3rd party tools where we probably
  can't add webhooks.
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

### Some more TODOs

* Support for creating/updating fly.io volumes
* Support for creation/updating fly.io secrets (not sure how though :S)
* More practical ways to configure Machine types, ram & cpu modifications
    * Right now it is possible, but only by setting the `launch_params` and/or `deploy_params` fields (see examples)
* better error handling :S
* better logging
* fly.io native postgres, redis, etc...

## Links/References

* [Git-Ops](https://www.redhat.com/en/topics/devops/what-is-gitops#:~:text=GitOps%20uses%20Git%20repositories%20as,set%20for%20the%20application%20framework.)
* [Argo-CD](https://argoproj.github.io/cd/)
* [GitHub webhooks](https://docs.github.com/en/webhooks-and-events/webhooks/about-webhooks)