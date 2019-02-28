# Docker Application Packages

An *experimental* utility to help make Compose files more reusable and sharable.

## CNAB support (preview)

You can find some preview binaries of `docker-app` with CNAB support [here](https://github.com/docker/app/releases/tag/cnab-dockercon-preview).

There is a [simple example](./examples/cnab-simple) and an [example of how to deploy a Helm Chart](./examples/cnab-helm).

## The problem application packages solve

Compose files do a great job of describing a set of related services. Not only are Compose files easy to write, they are generally easy to read as well. However, a couple of problems often emerge:

1. You have several environments where you want to deploy the application, with small configuration differences
2. You have lots of similar applications

Fundamentally, Compose files are not easy to share between concerns. Docker Application Packages aim to solve these problems and make Compose more useful for development _and_ production.

## Looking at an example

Let's take the following Compose file. It launches an HTTP server which prints the specified text when hit on the configured port.

```yaml
version: '3.2'
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "hello world"]
    ports:
      - 5678:5678
```

With `docker-app` [installed](#installation) let's create an Application Package based on this Compose file:

```bash
$ docker-app init --single-file hello
$ ls
docker-compose.yml
hello.dockerapp
```

We created a new file `hello.dockerapp` that contains three YAML documents:
- metadata
- the Compose file
- parameters for your application

It should look like this:

```yaml
version: 0.1.0
name: hello
description: ""
maintainers:
- name: yourusername
  email: ""

---
version: '3.2'
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "hello world"]
    ports:
      - 5678:5678

---
{}
```

Let's edit the parameters section and add the following default values for our application:

```yaml
port: 5678
text: hello development
version: latest
```

Then modify the Compose file section in `hello.dockerapp`, adding in the variables.

```yaml
version: '3.2'
services:
  hello:
    image: hashicorp/http-echo:${version}
    command: ["-text", "${text}"]
    ports:
      - ${port}:5678
```

Finally you can test everything is working, by rendering the Compose file with the provided default values.

```
$ docker-app render
version: "3.2"
services:
  hello:
    command:
    - -text
    - hello development
    image: hashicorp/http-echo:latest
    ports:
    - mode: ingress
      target: 5678
      published: 5678
      protocol: tcp
```

You can then use that Compose file like any other. You could save it to disk or pipe it straight to `docker stack` or `docker-compose` to launch the application.

```
$ docker-app render | docker-compose -f - up
```

This is where it gets interesting. We can override those parameters at runtime, using the `--set` option. Let's specify different option and run `render` again:

```
$ docker-app render --set version=0.2.3 --set port=4567 --set text="hello production"
version: "3.2"
services:
  hello:
    command:
    - -text
    - hello production
    image: hashicorp/http-echo:0.2.3
    ports:
    - mode: ingress
      target: 5678
      published: 4567
      protocol: tcp
```

If you prefer you can create a standalone configuration file to store those parameters. Let's create `prod.yml` with the following contents:

```yaml
version: 0.2.3
text: hello production
port: 4567
```

You can then run using that configuration file like so:

```
$ docker-app render -f prod.yml
```

More examples are available in the [examples](examples) directory.

## CNAB

Under the hood `docker-app` is CNAB compliant. It generates a CNAB from your application source and is able to install and manage any other CNAB too.
CNAB specifies three actions which `docker-app` provides as commands:
- `install`
- `upgrade`
- `uninstall`

**Note**: These commands need a Docker Context so that `docker-app` knows which endpoint and orchestrator to target.

```console
$ docker context create swarm --description "swarm context" --default-stack-orchestrator=swarm --docker=host=unix:///var/run/docker.sock
swarm
Successfully created context "swarm"

$ docker context ls
NAME                DESCRIPTION                               DOCKER ENDPOINT               KUBERNETES ENDPOINT   ORCHESTRATOR
default             Current DOCKER_HOST based configuration
swarm *             swarm context                             unix:///var/run/docker.sock                         swarm
```

Here is an example installing an application package, querying a status and then uninstalling it:
```console
$ docker-app install examples/hello-world/hello-world.dockerapp --name hello --target-context=swarm
Creating network hello_default
Creating service hello_hello

$ export DOCKER_TARGET_CONTEXT=swarm

$ docker-app status hello
ID                  NAME                MODE                REPLICAS            IMAGE                        PORTS
0m1wn7jrgkgj        hello_hello         replicated          1/1                 hashicorp/http-echo:latest   *:8080->5678/tcp

$ docker-app uninstall hello
Removing service hello_hello
Removing network hello_default
```

## Installation

Pre-built binaries are available on [GitHub releases](https://github.com/docker/app/releases) for Windows, Linux and macOS.
Each tarball contains two binaries:
- `docker-app-plugin-{linux|macos|windows}` which is docker-app as a [docker cli plugin](https://github.com/docker/cli/issues/1534) 
- `docker-app-standalone-{linux|macos|windows}` which is docker-app as a standalone utility 

To use `docker-app` plugin, just type `docker app` instead of `docker-app` and all the examples will work the same way:
```bash
$ docker app version
Version:      v0.8
Git commit:   XXX
Built:        Wed Feb 27 12:37:06 2019
OS/Arch:      darwin/amd64
Experimental: off
Renderers:    none

$ docker-app version
Version:      v0.8
Git commit:   XXX
Built:        Wed Feb 27 12:37:06 2019
OS/Arch:      darwin/amd64
Experimental: off
Renderers:    none
```

### Linux or macOS

Download your OS tarball:
```bash
export OSTYPE="$(uname | tr A-Z a-z)"
curl -fsSL --output "/tmp/docker-app-${OSTYPE}.tar.gz" "https://github.com/docker/app/releases/download/v0.6.0/docker-app-${OSTYPE}.tar.gz"
tar xf "/tmp/docker-app-${OSTYPE}.tar.gz" -C /tmp/
```

To install `docker-app` as a standalone:
```bash
install -b "/tmp/docker-app-standalone-${OSTYPE}" /usr/local/bin/docker-app
```

To install `docker-app` as a docker cli plugin:
```bash
mkdir -p ~/.docker/cli-plugins && cp "/tmp/docker-app-plugin-${OSTYPE}" ~/.docker/cli-plugins/docker-app
```

### Windows

Download the Windows tarball:
```powershell
Invoke-WebRequest -Uri https://github.com/docker/app/releases/download/v0.6.0/docker-app-windows.tar.gz -OutFile docker-app.tar.gz -UseBasicParsing
tar xf "docker-app.tar.gz"
```

To install `docker-app` as a standalone, copy it somewhere in your path:
```powershell
cp docker-app-plugin-windows.exe PATH/docker-app.exe
```

To install `docker-app` as a docker cli plugin:
```powershell
New-Item -ItemType Directory -Path ~/.docker/cli-plugins -ErrorAction SilentlyContinue
cp docker-app-plugin-windows.exe ~/.docker/cli-plugins/docker-app.exe 
```


**Note:** To use Application Packages as images (i.e.: `save`, `push`, or `install` when package is not present locally) on Windows, one must be in Linux container mode.

## Single file or directory representation

If you prefer having the three core documents in separate YAML files, omit the `-s` / `--single-file` option to
the `docker-app init` command. This will create a directory instead of a single file, containing
`metadata.yml`, `docker-compose.yml` and `parameters.yml`.

Converting between the two formats can be achieved by using the `docker-app split` and `docker-app merge` commands.

Note that you cannot store attachments in the single file format. If you want to use attachments you should use the directory format.

## Attachments (Storing additional files)

If you want to store additional files in the application package, such as `prod.yml`, `test.yml` or other config files, use the directory format and simply place these files inside the *.dockerapp/ directory. These will be bundled into the package when using `docker-app push`.

## Sharing your application on the Hub

You can push any application to the Hub using `docker-app push`:

``` bash
$ docker-app push --tag myhubuser/myimage:latest
```

This command will push to the Hub an image named `myhubuser/myimage:latest`.

If you omit the `--tag myhubuser/myimage:latest` argument, this command uses the application `version` defined in `metadata.yml` as the tag.

All `docker-app` commands accept an image name as input, which means you can run on a different host:

``` bash
$ docker-app inspect myhubuser/myimage
```

## Next steps

We have lots of ideas for making Compose-based applications easier to share and reuse, and making applications a first-class part of the Docker toolchain. Please let us know what you think about this initial release and about any of the ideas below:

* Introducing environments to the parameters file
* Docker images which launch the application when run
* Built-in commands for running applications
* Saving required images into the application artifact to support offline installation
* Signing applications with notary

If you're interested in contributing to the project, jump to [BUILDING.md](BUILDING.md) and [CONTRIBUTING.md](CONTRIBUTING.md).

## Usage

```
$ docker-app

Usage:  docker-app [OPTIONS] COMMAND

Build and deploy Docker Application Packages.

Options:
  -c, --context string     Name of the context to use to connect to the daemon (overrides DOCKER_HOST env var and default context set with "docker context use")
  -D, --debug              Enable debug mode
  -H, --host list          Daemon socket(s) to connect to
  -l, --log-level string   Set the logging level ("debug"|"info"|"warn"|"error"|"fatal") (default "info")
      --tls                Use TLS; implied by --tlsverify
      --tlscacert string   Trust certs signed only by this CA (default "/[home]/.docker/ca.pem")
      --tlscert string     Path to TLS certificate file (default "/[home]/.docker/cert.pem")
      --tlskey string      Path to TLS key file (default "/[home]/.docker/key.pem")
      --tlsverify          Use TLS and verify the remote
  -v, --version            Print version information

Commands:
  bundle      Create a CNAB invocation image and bundle.json for the application.
  completion  Generates completion scripts for the specified shell (bash or zsh)
  init        Start building a Docker application
  inspect     Shows metadata, parameters and a summary of the compose file for a given application
  install     Install an application
  merge       Merge a multi-file application into a single file
  push        Push the application to a registry
  render      Render the Compose file for the application
  split       Split a single-file application into multiple files
  status      Get the installation status. If the installation is a docker application, the status shows the stack services.
  uninstall   Uninstall an application
  upgrade     Upgrade an installed application
  validate    Checks the rendered application is syntactically correct
  version     Print version information

Run 'docker-app COMMAND --help' for more information on a command.
```

## Shell completion

### Bash

Load the docker-app completion code for bash into the current shell:
```console
$ source <(docker-app completion bash)
```
Set the docker-app completion code for bash to autoload on startup in your ~/.bashrc, ~/.profile or ~/.bash_profile:
```console
source <(docker-app completion bash)
```
**Note**: `bash-completion` is needed.

### Zsh

Load the docker-app completion code for zsh into the current shell
```console
$ source <(docker-app completion zsh)
```
Set the docker-app completion code for zsh to autoload on startup in your ~/.zshrc
```console
source <(docker-app completion zsh)
```

## Experimental

Some commands are flagged as experimental and will remain in this state until they mature. These commands are only accessible using an experimental binary. Feel free to test these commands and give us some feedback!

See [BUILDING.md/Experimental](BUILDING.md#experimental).
