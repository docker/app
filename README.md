# Docker Application

A Docker CLI Plugin to configure, share and install applications:
* Extend [Compose files](https://docs.docker.com/compose/compose-file/) with metadata and parameters
* Re-use same application across multiple environments (Development/QA/Staging/Production)
* Multi orchestrator installation (Swarm or Kubernetes)
* Push/Pull/[Promotion](https://docs.docker.com/ee/dtr/user/promotion-policies/internal-promotion/)/[Signing](https://docs.docker.com/engine/security/trust/content_trust/) supported for application, with same workflow as images
* Fully [CNAB](https://cnab.io) compliant
* Full support of Docker Contexts

## The problem Application Packages solves

Compose files do a great job of describing a set of related services. Not only
are Compose files easy to write, they are generally easy to read as well.
However, a couple of problems often emerge:

1. You have several environments where you want to deploy the application, with small configuration differences
1. You have lots of similar applications

Fundamentally, Compose files are not easy to share between concerns. Docker
Application Packages aim to solve these problems and make Compose more useful
for development _and_ production.

## Looking at an example

Let's take the following Compose file. It launches an HTTP server which prints
the specified text when hit on the configured port.

```yaml
version: '3.2'
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "hello world"]
    ports:
      - 5678:5678
```

With `docker app` [installed](#installation) let's create an Application Package
based on this Compose file:

```console
$ docker app init hello
$ ls
docker-compose.yml
hello.dockerapp
```

We created a new folder `hello.dockerapp` that contains three YAML documents:
- metadata
- the Compose file
- parameters for your application

It should look like this:

```yaml
version: 0.1.0
name: hello
description: A simple text server
maintainers:
- name: yourusername
  email:
```

```yaml
version: '3.2'
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "hello world"]
    ports:
      - 5678:5678
```

And an empty `parameters.yml. Let's edit and add the following default values for our applicatoin

```yaml
port: 5678
text: hello development
```

Then modify the Compose file section in `hello.dockerapp`, adding in the
variables.

```yaml
version: '3.2'
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "${text}"]
    ports:
      - ${port}:5678
```

You can test everything is working, by inspecting the application definition.

```console
$ docker app inspect
hello 0.1.0

Maintained by: yourusername

A simple text server

Service (1) Replicas Ports Image
----------- -------- ----- -----
hello       1        5678  hashicorp/http-echo

Parameters (2) Value
-------------- -----
port           5678
text           hello development
```

You can render the application to a Compose file with the provided default
values.

```console
$ docker app render
version: "3.2"
services:
  hello:
    command:
    - -text
    - hello development
    image: hashicorp/http-echo
    ports:
    - mode: ingress
      target: 5678
      published: 5678
      protocol: tcp
```

You can then use that Compose file like any other. You could save it to disk or
pipe it straight to `docker stack` or `docker-compose` to run the
application.

```console
$ docker app render | docker-compose -f - up
```

This is where it gets interesting. We can override those parameters at runtime,
using the `--set` option. Let's specify some different options and run `render`
again:

```console
$ docker app render --set port=4567 --set text="hello production"
version: "3.2"
services:
  hello:
    command:
    - -text
    - hello production
    image: hashicorp/http-echo
    ports:
    - mode: ingress
      target: 5678
      published: 4567
      protocol: tcp
```

If you prefer you can create a standalone configuration file to store those
parameters. Let's create `prod.yml` with the following contents:

```yaml
text: hello production
port: 4567
```

You can then run using that configuration file like so:

```console
$ docker app render --parameters-file prod.yml
```

You can share your Application Package by pushing it to a container registry.

```console
$ docker app push --tag myrepo/hello:0.1.0
```

Others can then use your Application Package by specifying the registry tag.

```console
$ docker app inspect myrepo/hello:0.1.0
```

**Note**: Commands like `install`, `upgrade`, `render`, etc. can also be used
directly on Application Packages that are in a registry.

You can specify the Docker endpoint where an application is installed using a
context and the `--target-context` option. If you do not specify one, it will
use the currently active context.

```console
$ docker context create remote --description "remote cluster" --docker host=tcp://<remote-ip>:<remote-port>
Successfully created context "remote"

$ docker context ls
NAME                DESCRIPTION                               DOCKER ENDPOINT               KUBERNETES ENDPOINT                ORCHESTRATOR
default *           Current DOCKER_HOST based configuration   unix:///var/run/docker.sock   https://localhost:6443 (default)   swarm
remote              remote cluster                            tcp://<remote-ip>:<remote-port>

$ docker app install myrepo/hello:0.1.0 --target-context remote
...
```

More examples are available in the [examples](examples) directory.

## CNAB

Under the hood `docker app` is [CNAB](https://cnab.io) compliant. It generates a
CNAB from your application source and is able to install and manage any other
CNAB too. CNAB specifies three actions which `docker app` provides as commands:
* `install`
* `upgrade`
* `uninstall`

Here is an example installing an Application Package, querying its status and
then uninstalling it:
```console
$ docker app install examples/hello-world/example-hello-world.dockerapp --name hello
Creating network hello_default
Creating service hello_hello

$ docker app status hello
ID                  NAME                MODE                REPLICAS            IMAGE                        PORTS
0m1wn7jrgkgj        hello_hello         replicated          1/1                 hashicorp/http-echo:latest   *:8080->5678/tcp

$ docker app uninstall hello
Removing service hello_hello
Removing network hello_default
```

## Installation

**Note**: This requires a pre-release version (19.03.0 beta 1 or later) of the
[Docker CLI](https://download.docker.com).

Pre-built static binaries are available on
[GitHub releases](https://github.com/docker/app/releases) for Windows, Linux and
macOS. Each tarball contains two binaries:
* `docker-app-plugin-{linux|darwin|windows.exe}` which is a [Docker CLI plugin](https://github.com/docker/cli/issues/1534). **Note**: This requires a pre-release version of the Docker CLI
* `docker-app-standalone-{linux|darwin|windows.exe}` which is a standalone utility

To use the standalone version, use `docker-app` instead of `docker app` and all
the examples will work the same way.

### Linux or macOS

Download your OS tarball:
```console
export OSTYPE="$(uname | tr A-Z a-z)"
curl -fsSL --output "/tmp/docker-app-${OSTYPE}.tar.gz" "https://github.com/docker/app/releases/download/v0.8.0/docker-app-${OSTYPE}.tar.gz"
tar xf "/tmp/docker-app-${OSTYPE}.tar.gz" -C /tmp/
```

To install as standalone:
```console
install -b "/tmp/docker-app-standalone-${OSTYPE}" /usr/local/bin/docker-app
```

To install as a Docker CLI plugin:
```console
mkdir -p ~/.docker/cli-plugins && cp "/tmp/docker-app-plugin-${OSTYPE}" ~/.docker/cli-plugins/docker-app
```

### Windows

Download the Windows tarball:
```powershell
Invoke-WebRequest -Uri https://github.com/docker/app/releases/download/v0.8.0/docker-app-windows.tar.gz -OutFile docker-app.tar.gz -UseBasicParsing
tar xf "docker-app.tar.gz"
```

To install as standalone, copy it somewhere in your path:
```powershell
cp docker-app-standalone-windows.exe PATH/docker-app.exe
```

To install as a Docker CLI plugin:
```powershell
New-Item -ItemType Directory -Path ~/.docker/cli-plugins -ErrorAction SilentlyContinue
cp docker-app-plugin-windows.exe ~/.docker/cli-plugins/docker-app.exe
```

## Attachments (Storing additional files)

If you want to store additional files in the application package, such as
`prod.yml`, `test.yml` or other config files, use the directory format and
simply place these files inside the *.dockerapp/ directory. These will be
bundled into the package when using `docker app push`.

## Sharing your application on the Hub

You can push any application to the Hub (or any registry) using
`docker app push`:

```console
$ docker app push --tag myhubuser/myimage:latest
```

This command will push an image named `myhubuser/myimage:latest` to the Docker
Hub.

If you omit the `--tag myhubuser/myimage:latest` argument, this command uses the
application `name` and `version` defined in `metadata.yml` as the tag:
`name:version`.

All `docker app` commands accept an image name as input, which means you can run
on a different host:

```console
$ docker app inspect myhubuser/myimage
```

The first time a command is executed against a given image name the bundle is
pulled from the registry and put in the local bundle store. You can pre-populate
this store by running `docker app pull myhubuser/myimage:latest`. All commands
manipulating a package also accept a `--pull` flag to force pulling the bundle
from the registry, even if it is present in the local store. This can be useful
when you are repeatedly pushing a bundle on the same tag.

### Multi-arch applications

By default the `docker app push` command only pushes service images for the linux/amd64
platform to the Docker Hub. By using the the `--all-platforms` flag it is possible to push
the services images for all platforms:

```console
$ docker app push --all-platforms myhubuser/myimage
```

It is also possible to push only a limited subset of platforms with the `--platform` flag:

```console
$ docker app push --platform linux/amd64 --platform linux/arm64 --platform linux/arm/v7 myhubuser/myimage
```

## Next steps

If you're interested in contributing to the project, jump to
[BUILDING.md](BUILDING.md) and [CONTRIBUTING.md](CONTRIBUTING.md).

## Usage

```
$ docker app

Usage:  docker app COMMAND

A tool to build and manage Docker Applications.

Commands:
  bundle      Create a CNAB invocation image and `bundle.json` for the application
  completion  Generates completion scripts for the specified shell (bash or zsh)
  init        Initialize Docker Application definition
  inspect     Shows metadata, parameters and a summary of the Compose file for a given application
  install     Install an application
  list        List the installations and their last known installation result
  pull        Pull an application package from a registry
  push        Push an application package to a registry
  render      Render the Compose file for an Application Package
  status      Get the installation status of an application
  uninstall   Uninstall an application
  upgrade     Upgrade an installed application
  validate    Checks the rendered application is syntactically correct
  version     Print version information

Run 'docker app COMMAND --help' for more information on a command.
```

## Shell completion

### Bash

Load the completion code for bash into the current shell:
```console
$ source <(docker app completion bash)
```
Set the completion code for bash to autoload on startup in your ~/.bashrc, ~/.profile or ~/.bash_profile:
```console
source <(docker app completion bash)
```
**Note**: `bash-completion` is needed.

### Zsh

Load the completion code for zsh into the current shell
```console
$ source <(docker app completion zsh)
```
Set the completion code for zsh to autoload on startup in your ~/.zshrc
```console
source <(docker app completion zsh)
```

## Experimental

Some commands are flagged as experimental and will remain in this state until
they mature. These commands are only accessible using an experimental binary.
Feel free to test these commands and give us some feedback!

See [BUILDING.md/Experimental](BUILDING.md#experimental).
