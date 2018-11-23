# Docker App Package Specification

This section describes all the requirements for interoperability.

## YAML Documents

A Docker App Package is a set of 3 YAML documents:
* `metadata`
* `docker-compose`
* `parameters`

These documents can be split in 3 different files or merged into one YAML file, using the [multi document YAML feature](http://yaml.org/spec/1.2/spec.html#id2760395).
The order of the documents in a multi-documents YAML is **strict**:
1. metadata
1. docker-compose
1. parameters

### metadata.yml

`metadata.yml` defines some informations to describe the application in a standard `YAML` file.
See [JSON Schemas](schemas/) for validation.

### docker-compose.yml

`docker-compose.yml` is a standard [Compose file](https://docs.docker.com/compose/compose-file/) with variable replacement.
`Compose` minimum version is **v3.2**, see [JSON Schemas](https://github.com/docker/cli/tree/master/cli/compose/schema/data) for validation.

### parameters.yml

`parameters.yml` is a simple Key-Value file used to replace the variables defined in the `docker-compose` file. As it is an open document, there is no schema for this one.

## Validation

Use the `validate` command:
```
Checks the rendered application is syntactically correct

Options:
  -f, --parameters-file stringArray   Override with parameters from file
  -s, --set stringArray               Override parameters values
```

Here is an example:

```sh
# Init an empty docker application package, with an invalid mail for a maintainer
$ docker-app init my-app --maintainer "name:invalid#mail.com"
# Try to validate the application package
$ docker-app validate my-app
Error: failed to validate metadata:
- maintainers.0.email: Does not match format 'email'

# Fix the metadata file
$ vi my-app.dockerapp/metadata.yml
# And re-try validation
$ docker-app validate my-app
$ echo $?
0
```
