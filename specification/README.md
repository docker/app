# Docker App Specification

This section describes all the requirements for docker-app interoperability.

## YAML Documents

A Docker App is a set of 3 YAML documents:
* `metadata`
* `docker-compose`
* `settings`

These documents can be split in 3 different files or merged into one YAML file, using the [multi document YAML feature](http://yaml.org/spec/1.2/spec.html#id2760395).

### metadata.yml

`metadata.yml` defines some informations to describe the application in a standard `YAML` file.
See [JSON Schemas](schemas/) for validation.

### docker-compose.yml

`docker-compose.yml` is a standard [Compose file](https://docs.docker.com/compose/compose-file/) with variable replacement.
`Compose` minimal version is **v3.2**, see [JSON Schemas](https://github.com/docker/cli/tree/master/cli/compose/schema/data) for validation.

### settings.yml

`settings.yml` is a simple Key-Value file used to replace the variables defined in the `docker-compose` file. As it is an opened document, there is no schema for this one.

## Validation

The tool `yamlschema` included in `cmd/yamlschema` helps you validating a `YAML document` against its `JSON Schema`. 

Here is an example:

```sh
# Init an empty docker application
$ docker-app init my-app
# Build the YAML schema validator
$ make bin/yamlschema

# Validate the metadata.yml freshly created against its schema. It should fail as some information values are missing.
$ ./bin/yamlschema my-app.dockerapp/metadata.yml specification/schemas/metadata_schema_v0.2.json
The document is not valid. See errors :
- description: Invalid type. Expected: string, given: null

$ echo $?
1

# Fill the missing parts
$ vi my-app.dockerapp/metadata.yml
# ... and re-invoke the validator
$ cat my-app.dockerapp/metadata.yml | ./bin/yamlschema - schema/data/metadata_schema_v0.2.json
$ echo $?
0

# Now edit your docker-compose.yml
$ vi my.app.dockerapp/docker-compose.yml
# ... and validate it against the compose schema from the docker/cli
$ ./bin/yamlschema my-app.dockerapp/docker-compose.yml https://raw.githubusercontent.com/docker/cli/master/cli/compose/schema/data/config_schema_v3.2.json
$ echo $?
0
```

## Helm

`docker-app` can produce a [Helm Chart v2](https://docs.helm.sh/developing_charts/#charts) from a `Docker Application`. The chart needs a Kubernetes Compose component to work, which is included by default in `Docker for Desktop` and `Docker Entreprise Edition`. It uses by default the **v1beta2** `Stack resource` definition, but can also handle the **v1beta** version. 
