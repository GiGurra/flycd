# FlyCD

## Concept

![alt text](https://raw.githubusercontent.com/GiGurra/flycd/master/concept.svg)


FlyCD is a tool designed to add ArgoCD/Flux style git-ops support for Fly.io. Although its fully automated git-ops
functionality is still to be implemented, the following are the features it aims to provide:

* Extending the regular fly.io fly.toml specifications with additional configuration parameters,
  removing the need for running **_any_** flyctl commands manually.

* FlyCD separates app development from app environment deployment/composition.
    * You write code in one repo and push updates to your app. Don't include any environment specific configuration in
      your app repo.
    * Have arbitrary number of fly.io environments making use of that app, in different
      versions, configurations etc. No need to embed environment specific configurations into your app.
    * Reference/deploy any app, from your own repos or repos owned by others, and compose the cloud environment you want
      separately from app development.

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
    * figures out if a deployment is actually warranted, by configuration folder and app folder checksum/app git hash
      with hashes saved to fly.io env for the app (using app env vars for this)
* It can deploy many apps at the same time. Simply point it to a directory structure/hierarchy containing multiple
  app.yaml files, and flycd will traverse the structure recursively, clone each app's source repo and deploy each app
* It can currently install itself into an existing fly.io environment
* Needs some persistence and queueing of incoming webhook commands to no run into data races, or lose data if flycd
  crashes or is re-deployed :S. Currently, it just spins up a go-routine for running the new deployment.
* Needs a better way to scan existing available apps than re-read all of their specs from disk on every webhook event.
* Need separate config repo(s) and webhooks. Right now flycd when deployed needs locally mounted projects/ folder.
* Need regular jobs/auto sync for apps that don't send webhooks, like 3rd party tools where we probably can't add
  webhooks.
* Need some security validation of webhooks from GitHub :D. Currently, there is none so DOS attacks are trivial to create :S. 

**I have no idea if I will have time or interest in continuing this project until it reaches a useful state :D.**
Consider it proof of concept, and nothing more.

## Current issues

* SUPER HACKY code right now, just a 3-day hack so far with most work delegated to shell commands instead of proper go
  libraries :D
    * Lots of refactoring needed!
* This functionality might already exist/I might be reinventing the wheel here - we will see what is written in the
  discussion thread over at fly.io community forums.
    *
  see https://community.fly.io/t/simple-self-contained-argocd-style-git-ops-with-fly-io-what-are-the-options-poc-flycd-app/14032
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

## Using it

### Getting started

1. Fork this repo
2. Modify the contents of the `project` folder (or create one), and add the app specifications you like
3. Run `go run . deploy projects` to ensure it deploys things the way you expect
4. Run `go run . install <your-preferred-flycd-app-name> <org> <region>` to install flycd into your fly.io environment.
   This will create a new fly.io app with <your-preferred-flycd-app-name> running flycd in monitoring mode/webhook
   listening mode. There are some env vars you can use to modify the webhook path. The `install` command will
   automatically issue a fly.io API token for itself, and store it as an app secret in fly.io. You can ssh into your
   flycd container and copy it from there if you want to use it for other purposes (you prob shouldn't) or just locally
   verify that it works.
5. Add a webhook to your git repo, pointing to your flycd app's url,
   e.g. the default POST path `https://<your-flycd-app-name>.fly.dev/webhook`, which currently just supports GitHub push
   webhooks.
6. Watch the magic happen!

## Sample project structure

Replace the `projects/` folder in your fork with something more useful.

```
projects/
└── example-project1
    ├── local_app
    │   ├── Dockerfile
    │   └── app.yaml
    └── x_git_app
        └── app.yaml
```

Check the examples in here to see how to configure your apps, or just check my hacky code :D