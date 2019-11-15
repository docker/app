# Example: Hello World

In this example, we will create a Docker App which is a single service application deploying a web
server with a configurable text message.

## Creating an App definition

First, we create an App definition using the `docker app init` command:

```shell
$ docker app init hello-world
Created "hello-world.dockerapp"
$ tree
.
├── hello-world.dockerapp
    ├── docker-compose.yml
    ├── metadata.yml
    └── parameters.yml
```

A new folder named `hello-world.dockerapp` now exists, which contains three YAML documents:
* metadata
* a [Compose file](https://docs.docker.com/compose/compose-file/)
* parameters to be used at runtime

The `metadata.yml` file should display as follows:

```yaml
# Version of the application
version: 0.1.0
# Name of the application
name: hello-world
# A short description of the application
description:
# List of application maintainers with name and email for each
maintainers:
  - name: user
    email:
```

The `docker-compose.yml` should contain the following:

```yaml
version: "3.6"
services: {}
```

The `parameters.yml`file should be empty.

## Editing App definition

Open `hello-world.dockerapp` with your favorite text editor.

### Editing the metadata

Open the `metadata.yml` file and edit the `description` and `maintainers` fields in the metadata section.

### Editing the list of services

Open the `docker-compose.yml` file and add a `hello` service to the `services` section.

```yaml
version: "3.6"
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "${text}"]
    ports:
      - ${port}:5678
```

### Editing the parameters

In the `parameters.yml` file, add variables with their default value:

```yaml
port: 8080
text: Hello, World!
```

## Building an App image

Next, build an App image from the App definition we have created:

```shell
$ docker app build . -f hello-world.dockerapp -t myrepo/hello:0.1.0
[+] Building 0.6s (6/6) FINISHED
(...) (Build output)
sha256:7b48c121fcafa0543b7e88c222304f9fada9911011694b041a7f0e096536db6c
```

At this point, an App image with the `myrepo/hello:1.0.1` tag has been built from the `hello-world.dockerapp` App definition. This immutable App image includes all the service images at fixed versions that you can run or share.

## Inspecting an App image

Now let's get detailed information about the App image we just built using the `docker app image inspect` command. Note that the `--pretty` option allows to get a human friendly output rather than the JSON default output.

```shell
$ docker  app image inspect myrepo/hello-world:0.1.0 --pretty
version: 0.1.0
name: hello-world
description: This is an Hello World example
maintainers:
- name: user
  email: user@email.com

SERVICE   REPLICAS   PORTS   IMAGE
hello     1          8080    docker.io/hashicorp/http-echo:latest@sha256:ba27d460cd1f22a1a4331bdf74f4fccbc025552357e8a3249c40ae216275de96

PARAMETER VALUE
port      8080
text      Hello, World!
```

## Sharing the App

Share your App image by pushing it to a container registry such as Docker Hub.

```shell
$ docker app push myrepo/hello:0.1.0
```

## Running the App

Now run your App:

```shell
$ docker app run myrepo/hello:0.1.0  --name myhelloworld
Creating network myhelloworld_default
Creating service myhelloworld_hello
App "myhelloworld" running on context "default"
```

You can specify the Docker endpoint where an application is installed using a context. By default, your App will run on the currently active context. You can select another context with the docker context use command, and the docker app run command will thereafter run your app on this particular context.

Next, you can check the list of running Apps:

```shell
$ docker app ls
INSTALLATION   APPLICATION         LAST ACTION   RESULT    CREATED              MODIFIED             REFERENCE
myhelloworld   hello-world (0.1.0) install       success   About a minute ago   About a minute ago   docker.io/myrepo/hello-world:0.1.0
```

## Inspecting a running App

Finally you can get detailed information about a running App using the `docker app inspect` command. Note that the `--pretty` option allows to get a human friendly output rather than the JSON default output.

```shell
$ docker app inspect myhelloworld --pretty
Installation:
  Name: myhelloworld
  Created: 3 minutes ago
  Modified: 3 minutes ago
  Revision: 01DSQMWWABCM27K3FWSCES3H76
  Last Action: install
  Result: success
  Ochestrator: swarm

Application:
  Name: hello-world
  Version: 0.1.0
  Image Reference: docker.io/myrepo/hello-world:0.1.0

Parameters:
  port: "8080"
  text: Hello, World!

ID            NAME                MODE        REPLICAS  IMAGE                PORTS
mocfqnkadxw3  myhelloworld_hello  replicated  1/1       hashicorp/http-echo  *:8080->5678/tcp
```