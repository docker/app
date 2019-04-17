## Hello world!

### Initialize project

In this example, we will create a single service application that deploys a web
server with a configurable text message.

First, we will initialize the project using the single file format. A single
file application contains three sections: `metadata` which corresponds to
`metadata.yml`, `parameters` which corresponds to `parameters.yml` and
`services` which corresponds to `docker-compose.yml`.

```console
$ docker app init --single-file hello-world
$ ls -l
-rw-r--r-- 1 README.md
-rw-r--r-- 1 example-hello-world.dockerapp
$ cat hello-world.dockerapp
# This section contains your application metadata.
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

---
# This section contains the Compose file that describes your application services.
version: "3.6"
services: {}

---
# This section contains the default values for your application parameters.
```

Open `hello-world.dockerapp` with your favorite text editor.

### Edit metadata

Edit the `description` and `maintainers` fields in the metadata section.

### Edit the services

Add a service `hello` to the `services` section.

```yaml
[...]
---
# This section contains the Compose file that describes your application services.
version: "3.6"
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "${text}"]
    ports:
      - ${port}:5678

---
[...]
```

### Edit the parameters

In the parameters section, add every variables with the default value you want,
e.g.:

```yaml
[...]
---
# This section contains the default values for your application parameters.
port: 8080
text: Hello, World!
```

### Inspect

Inspecting a Docker Application gives you a summary of what the application
includes.

```console
$ docker app inspect hello-world.dockerapp
hello-world 0.1.0

Maintained by: user <user@email.com>

Hello, World!

Service (1) Replicas Ports Image
----------- -------- ----- -----
hello       1        8080  hashicorp/http-echo

Parameters (2) Value
-------------- -----
port           8080
text           Hello, World!
```

### Render

You can render your application to a Compose file  by running
`docker app render` or even personalize the rendered Compose file by running
`docker app render --set text="hello user!"`.

Create a `render` directory and redirect the output of `docker app render ...`
to `render/docker-compose.yml`:

```console
$ mkdir render
$ docker app render --set text="Hello user!" > render/docker-compose.yml
```

### Install

You directly install your application by running
`docker app deploy --set text="Hello user!"` or you can use the rendered version
and run `docker stack deploy render/docker-compose.yml`.

Navigate to `http://<ip_of_your_node>:8080` with a web browser and you will see
the text message. Note that `<ip_of_your_node>` is `127.0.0.1` if you installed
to your local Docker endpoint.

```console
$ curl 127.0.0.1:8080
Hello user!
```

### Push

You can share your application by pushing it to a container registry such as
the Docker Hub.

```console
$ docker app push hello-world --tag myrepo/hello-world:0.1.0
```

You can then use your application package directly from the repository:

```console
$ docker app inspect myrepo/hello-world:0.1.0
hello-world 0.1.0

Maintained by: user <user@email.com>

Hello, World!

Service (1) Replicas Ports Image
----------- -------- ----- -----
hello       1        8080  myrepo/hello-world@sha256:ba27d460cd1f22a1a4331bdf74f4fccbc025552357e8a3249c40ae216275de96

Parameters (2) Value
-------------- -----
port           8080
text           Hello, World!
```
