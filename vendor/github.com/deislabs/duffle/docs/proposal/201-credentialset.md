# Duffle Credential Sets

This document covers how credentials are passed into Duffle from the environment.

## The Credential Problem

Consider the case where a CNAB bundle named `example/myapp:1.0.0` connects to both ARM (Azure IaaS API service) and Kubernetes. Each has its own API surface which is secured by a separate set of credentials. ARM requires a periodically expiring token managed via `az`. Kubernetes stores credentialing information in a `KUBECONFIG` YAML file.

When the user runs `duffle install my_example example/myapp:1.0.0`, the operations in the invocation image need to be executed with a specific set of credentials from the user.

Those values must be injected from `duffle` into the invocation image.

But Duffle must know which credentials to send.

The situation is complicated by the following factors:

- There is no predefined set of services (and thus predefined set of credentialing) specified by CNAB.
- It is possible, and even likely, that a user may have more than one set of credentials for a service or backend (e.g. credentials for two different Kubernetes clusters)
- Some credentials require commands to be executed on a host, such as unlocking a vault or regenerating a token
- There is no standard file format for storing credentials
- The consuming applications may require the credentials be submitted via different methods, including as environment variables, files, or STDIN.

Subsequently, any satisfactory solution must be able to accommodate a wide variety of configurational permutations, ideally without dictating that credentialing tools change in any way.

## Credential Sets

A *credential set* is a named set of credentials (or credential generators) that is managed on the local host (via duffle) and injected into the invocation container on demand.

### On-disk format

The `$HOME/.duffle/` directory is where user-specific Duffle configuration files are placed. Inside of this directory, Duffle stores credential information in its own subdirectory:

```bash
$HOME/.duffle/
  |- credentials/
          |- production.yaml
          |- staging.yaml
```

NOTE: YAML is not a required format, but it's easy to write as a real human. So... I'll start there

A credential YAML file contains a set of named credentials that are resolved locally (if necessary) and then pushed into the container.

Example (`staging.yaml`):

```yaml
name: staging     # Must match the name portion of the file name (staging.yaml)
credentials:
  - name: read_file
    source:
      path: $SOMEPATH/testdata/someconfig.txt  # credential will be read from this file
                                               # In 'path', env vars are evaluated.
  - name: run_program
    source:
      command: "echo wildebeest" # The command `echo wildebeest` will be executed
                                 # An error will cause the process to exit
  - name: use_var
    source:
      env: TEST_USE_VAR      # This will read an env var from local, and copy to dest
      value: "this space intentionally left non-blank"
  - name: fallthrough
    source:
      name: NO_SUCH_VAR      # Assuming this is not set....
      value: quokka          # Then this will be used as the default value
  - name: plain_value
    source:
      value: cassowary       # Load this literal value.
```

The above shows several examples of how credentials can be loaded from a local source and
sent to an in-image destination.

Loading from source is done from four potential inputs:

- `value` is a literal value
- `env` is loaded from an environment variable (and can fall back to `value` as a default)
- `path` is loaded from a file at the given path (or else it errors)
- `command` executes a command, and returns the output as the value (or else it errors)

Duffle will capture (at runtime) the data presented by these sources, and will pass the data into the container as required.

Credential sets are specified when needed:

```console
$ duffle install --credentials=staging my_example example/myapp:1.0.0
> loading credentials from $HOME/.duffle/credentials/staging.yaml
> running example/myapp:1.0.0
```

Credential sets are loaded locally. All commands are executed locally. Then the results are injected into the image at startup.

## Matching Credentials in a Bundle

Bundles declare which credentials they require. This information is specified in the `bundle.json`:

```json
{
    "schemaVersion": 1,
    "name": "helloworld",
    "version": "0.1.2",
    "description": "An example 'thin' helloworld Cloud-Native Application Bundle",
    "invocationImages":[],
    "images": [],
    "parameters": {},
    "credentials": {
        "kubeconfig": {
            "path": "/home/.kube/config",
        },
        "image_token": {
            "env": "AZ_IMAGE_TOKEN",
        },
        "hostkey": {
            "path": "/etc/hostkey.txt",
            "env": "HOST_KEY"
        }
    }
}
```

The `credentials` section maps a name (e.g. `kubeconfig`) to the destination (e.g. `path: ...` or `env: ...`).

Duffle will match the credentials requested in the `bundle.json` to the credentials specified in the credential set passed with the `--credentials` flag. Matching is done by name. Thus, to send configuration data to the above bundle, we would need a credentialset like this:

```yaml
name: mycreds
credentials:
  - name: kubeconfig
    source:
      path: $HOME/.kube/config
  - name: image_token
    source:
      value: "abcdefg"
  - name: hostkey
    source:
      env: "HOSTKEY"
  - name: sir-not-appearing-in-this-film
    source:
      value: unused
```

When the above bundle (`helloworld`) is installed with the above credentials (`mycreds`), the credentials are resolved as follows:

- `kubeconfig` is read from Duffle's local path (`$HOME/.kube/config`), and the contents are placed into the invocation image at the path `/home/.kube/config`
- `image_token` is treated as a literal value, and the string `abcdefg` is injected into the invocation image as the environment variable `$AZ_IMAGE_TOKEN`
- `hostkey` is read from the local environment variable `$HOSTKEY`, and is then injected into two places in the invocation image:
  - It is set as the value of `$HOST_KEY`
  - It is placed in the file `/etc/hostkey.txt`

Since the last credential in the credential set (`sir-not-appearing-in-this-film`) is not required by the bundle, it is ignored.

During the resolution phase, _if any required credential is not provided in the credential set, the operation is aborted and Duffle exits with a failure._

## Limitations

In this model, credentials can only be injected as files and environment variables. Some systems may not be satisfied with this limitation, in which case additional scripting may be required inside of the invocation image.

Next Section: [drivers](202-drivers.md)