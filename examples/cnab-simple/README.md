# Docker Application to CNAB

### Requirements

* [Docker Desktop](https://www.docker.com/products/docker-desktop) with Kubernetes enabled or any other Kubernetes cluster
* Source code from this directory

### Examples

Show the details of the application with `inspect`

```console
$ docker app inspect
hello 0.2.0

Maintained by: garethr <someone@example.com>

Sample app for DockerCon EU 2018

Service (1) Replicas Ports Image
----------- -------- ----- -----
hello       1        8765  hashicorp/http-echo:0.2.3

Parameters (2) Value
-------------- -----
port           8765
text           Hello DockerCon!
```

Install the application:

```console
$ docker app install
```

Update the installation, demonstrating setting parameters:

```console
$ docker app update --set port=9876 --set text="hello DockerCon EU" hello
```

Uninstall the application installation:

```console
$ docker app uninstall hello
```

Demonstrate building a `bundle.json` for CNAB.

```console
$ docker app bundle
Invocation image "hello:0.2.0-invoc" successfully built
$ cat bundle.json
{
  "name": "hello",
  "version": "0.2.0",
  "description": "Sample app for DockerCon EU 2018",
  ...
}
```
