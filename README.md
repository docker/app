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
targets:
  swarm: true
  kubernetes: true

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
wget https://github.com/docker/app/releases/download/v0.2.0/docker-app-linux.tar.gz
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

If you prefer having the three documents in separate YAML files, omit the `-s` option to
the `docker-app init` command. This will create a directory instead of a singe file, containing
`metadata.yml`, `docker-compose.yml` and `settings.yml`.

## Sharing your application on the Hub

You can push any application to the Hub using `docker-app push`:

``` bash
$ docker-app push --namespace myHubUser --tag latest
```

This command will create an image named `myHubUser/hello.dockerapp:latest` on your local Docker
daemon, and push it to the Hub.

By default, this command uses the application version defined in `metadata.yml` as the tag,
and the value of the metadata field `namespace` as the image namespace.

All `docker-app` commands accept a local image name as input, which means you can run on a different host:

``` bash
$ docker pull myHubUser/hello.dockerapp:latest
$ docker-app inspect myHubUser/hello
```

## Next steps

We have lots of ideas for making Compose-based applications easier to share and reuse, and making applications a first-class part of the Docker toolchain. Please let us know what you think about this initial release and about any of the ideas below:

* Introducing environments to the settings file
* Docker images which launch the application when run
* Built-in commands for running applications
* Saving required images into the application artifact to support offline installation
* Automatically validating Compose files against the schema for the specified version
* Signing applications with notary


## Usage

```
$ docker-app
Build and deploy Docker applications.

Usage:
  docker-app [command]

Available Commands:
  deploy      Deploy or update an application
  helm        Generate a Helm chart
  help        Help about any command
  init        Start building a Docker application
  inspect     Shows metadata and settings for a given application
  ls          List applications.
  push        Push the application to a registry
  render      Render the Compose file for the application
  save        Save the application as an image to the docker daemon(in preparation for push)
  version     Print version information

Flags:
      --debug   Enable debug mode
  -h, --help    help for docker-app

Use "docker-app [command] --help" for more information about a command.
```
