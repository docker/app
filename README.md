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

With `docker-app` installed let's create an Application Package based on this Compose file:

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
namespace: ""
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

## Installation

Pre-built binaries are available on [GitHub releases](https://github.com/docker/app/releases) for Windows, Linux and macOS.

```bash
wget https://github.com/docker/app/releases/download/v0.6.0/docker-app-linux.tar.gz
tar xf docker-app-linux.tar.gz
cp docker-app-linux /usr/local/bin/docker-app
```

**Note:** To use Application Packages as images (i.e.: `save`, `push`, or `deploy` when package is not present locally) on Windows, one must be in Linux container mode.

## Single file or directory representation

If you prefer having the three core documents in separate YAML files, omit the `-s` / `--single-file` option to
the `docker-app init` command. This will create a directory instead of a single file, containing
`metadata.yml`, `docker-compose.yml` and `parameters.yml`.

Converting between the two formats can be achieved by using the `docker-app split` and `docker-app merge` commands.

Note that you cannot store attachments in the single file format. If you want to use attachments you should use the directory format.

## Attachments (Storing additional files)

If you want to store additional files in the application package, such as `prod.yml`, `test.yml` or other config files, use the directory format and simply place these files inside the *.dockerapp/ directory. These will be bundled into the package when using `docker-app push`

## Sharing your application on the Hub

You can push any application to the Hub using `docker-app push`:

``` bash
$ docker-app push --namespace myhubuser --tag latest
```

This command will push to the Hub an image named `myhubuser/hello.dockerapp:latest`.

If you omit the `--tag latest` argument, this command uses the application `version` defined in `metadata.yml` as the tag.
If you omit the `--namespace myhubuser` argument, this command uses the application `namespace` defined in `metadata.yml` as the image namespace.

All `docker-app` commands accept an image name as input, which means you can run on a different host:

``` bash
$ docker-app inspect myhubuser/hello
```

## Next steps

We have lots of ideas for making Compose-based applications easier to share and reuse, and making applications a first-class part of the Docker toolchain. Please let us know what you think about this initial release and about any of the ideas below:

* Introducing environments to the parameters file
* Docker images which launch the application when run
* Built-in commands for running applications
* Saving required images into the application artifact to support offline installation
* Signing applications with notary


## Usage

```
$ docker-app

Usage:  docker-app [OPTIONS] COMMAND

Build and deploy Docker Application Packages.

Options:
  -c, --context string     context to use to connect to the daemon (overrides host flag, DOCKER_HOST env var and default context set with "docker context use")
  -D, --debug              Enable debug mode
  -H, --host list          Daemon socket(s) to connect to
  -l, --log-level string   Set the logging level ("debug"|"info"|"warn"|"error"|"fatal") (default "info")
      --tls                Use TLS; implied by --tlsverify
      --tlscacert string   Trust certs signed only by this CA (default "/Users/chris/.docker/ca.pem")
      --tlscert string     Path to TLS certificate file (default "/Users/chris/.docker/cert.pem")
      --tlskey string      Path to TLS key file (default "/Users/chris/.docker/key.pem")
      --tlsverify          Use TLS and verify the remote
  -v, --version            Print version information

Commands:
  add-credentialset Add a CNAB credentialset in the credential store for the given Docker Context
  bundle            Create a CNAB invocation image and bundle.json for the application.
  completion        Generates completion scripts for the specified shell (bash or zsh)
  init              Start building a Docker application
  inspect           Shows metadata, parameters and a summary of the compose file for a given application
  install           Install an application
  merge             Merge a multi-file application into a single file
  pull              Pull an application from a registry
  push              Push the application to a registry
  render            Render the Compose file for the application
  split             Split a single-file application into multiple files
  status            Get an application status
  uninstall         Uninstall an application
  validate          Checks the rendered application is syntactically correct
  version           Print version information

Run 'docker-app COMMAND --help' for more information on a command.
```

## Shell completion

### Bash

Load the docker-app completion code for bash into the current shell:
```sh
$ source <(docker-app completion bash)
```
Set the docker-app completion code for bash to autoload on startup in your ~/.bashrc, ~/.profile or ~/.bash_profile:
```sh
source <(docker-app completion bash)
```
**Note**: `bash-completion` is needed.

### Zsh

Load the docker-app completion code for zsh into the current shell
```sh
$ source <(docker-app completion zsh)
```
Set the docker-app completion code for zsh to autoload on startup in your ~/.zshrc
```sh
source <(docker-app completion zsh)
```
