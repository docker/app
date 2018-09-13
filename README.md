# Docker Application Packages

An *experimental* utility to help make Compose files more reusable and sharable.


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
- metadatas
- the Compose file
- settings for your application

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

Let's edit the settings section and add the following default values for our application:

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

This is where it gets interesting. We can override those settings at runtime, using the `--set` option. Let's specify different option and run `render` again:

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

If you prefer you can create a standalone configuration file to store those settings. Let's create `prod.yml` with the following contents:

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
wget https://github.com/docker/app/releases/download/v0.5.0/docker-app-linux.tar.gz
tar xf docker-app-linux.tar.gz
cp docker-app-linux /usr/local/bin/docker-app
```

**Note:** To use Application Packages as images (i.e.: `save`, `push`, or `deploy` when package is not present locally) on Windows, one must be in Linux container mode.

## Integrating with Helm

`docker-app` comes with a few other helpful commands as well, in particular the ability to create Helm Charts from your Docker Applications. This can be useful if you're adopting Kubernetes, and standardising on Helm to manage the lifecycle of your application components, but want to maintain the simplicity of Compose when writing you applications. This also makes it easy to run the same applications locally just using Docker, if you don't want to be running a full Kubernetes cluster.

```
$ docker-app helm
```

This will create a folder, `<my-application-name>.chart`, in the current directory. The folder contains the required `Chart.yaml` file and templates describing the `stack` Kubernetes object based on the Compose file in your application.

_Note that this requires the Compose Kubernetes controller available in Docker for Windows and Docker for Mac, and in Docker Enterprise Edition._

### Helm chart for Docker EE 2.0

In order to create a helm chart that is compatible with version 2.0 of Docker Enterprise Edition, you will need to use the `--stack-version` flag to create a compatible version of the helm chart using `v1beta1` like so:

```bash
$ docker-app helm --stack-version=v1beta1
```

## Single file or directory representation

If you prefer having the three core documents in separate YAML files, omit the `-s` / `--single-file` option to
the `docker-app init` command. This will create a directory instead of a single file, containing
`metadata.yml`, `docker-compose.yml` and `settings.yml`.

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

## Forking an existing image

Found an app on a remote registry you'd like to modify to better suit your needs? Use the `fork` subcommand:

```bash
$ docker-app fork remote/hello.dockerapp:1.0.0 mine/hello2 -m "Bob Dylan:bob@aol.com"
```

This command will create a local, editable copy of the app on your system. By default, the copy is created inside the current directory; you may use the `--path` flag to configure a different destination.

For example, the following will create the `/opt/myapps/hello2.dockerapp` folder containing the forked app's files:

```bash
$ docker-app fork remote/hello.dockerapp:1.0.0 mine/hello2 --path /opt/myapps
```

## Storing images in the application package

Using the `docker-app image add` command, one can store all images used by the
application into the `images` subfolder.

Those images can then be pushed to another registry using `docker-app image push`. Finally the
`render` and `deploy` commands support overriding the image names with any registry using the `--registry` argument.

What follows is a complete deployment scenario of an application from the Hub into a cluster without access
to the Internet:

``` sh
# Download myapp from the Hub locally
opensystem$ docker-app split myuser/myapp.dockerapp
# Store used images into the application folder.
# The images must be available on your local docker daemon
opensystem$ docker-app image add myapp
# Now transport the myapp.dockerapp folder to the air-gapped cluster and login:
# Push the application's images to a local registry
closed$ docker-app image push myapp myregistry:5000
# Deploy overriding image names with the local registry
closed$ docker-app deploy --registry myregistry:5000 myapp
```

## Next steps

We have lots of ideas for making Compose-based applications easier to share and reuse, and making applications a first-class part of the Docker toolchain. Please let us know what you think about this initial release and about any of the ideas below:

* Introducing environments to the settings file
* Docker images which launch the application when run
* Built-in commands for running applications
* Signing applications with notary


## Usage

```
$ docker-app

Usage:  docker-app [OPTIONS] COMMAND

Docker Application Packages

Options:
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
  completion  Generates completion scripts for the specified shell (bash or zsh)
  deploy      Deploy or update an application
  fork        Create a fork of an existing application to be modified
  helm        Generate a Helm chart
  init        Start building a Docker application
  inspect     Shows metadata, settings and a summary of the compose file for a given application
  merge       Merge a multi-file application into a single file
  push        Push the application to a registry
  render      Render the Compose file for the application
  split       Split a single-file application into multiple files
  validate    Checks the rendered application is syntactically correct
  version     Print version information

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
