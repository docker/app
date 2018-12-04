# Duffle Drivers

Duffle can perform various actions, such as `install` and `uninstall`. These actions interact with CNAB images. For example, a `duffle install my_app example:1.2.3` will execute the invocation image for the CNAB `example:1.2.3`.

It is possible that users may want to determine which runtime is used to execute the invocation image. For example, if the image is a Docker image, a user may prefer to run it with Docker, or they may prefer to run it with an alternative client (like `rkt`), or they may prefer to execute it within a cloud service like Azure Container Instances.

To accommodate this case, Duffle provides multiple drivers.

## Image Types

Duffle inspects the bundle.json to determine the _image type_ of the image. (FIXME: Actually, right now it just assumes Docker. See `cmd/duffle/install.go`.) Each image can be a particular type, such as a Docker image or a QCOW image.

Drivers can support different image types. The `docker` driver supports `oci` and `docker` image types. The `debug` driver supports all image types. When creating a new driver, developers must specify which image types that driver can support.

## Requesting a Driver

By default, the `docker` driver is used. But a user may choose to override the driver by specify the `-d DRIVERNAME` flag on the relevant operation.

A driver will (MUST) fail if the given driver cannot handle the CNAB bundle's invocation image.

## Built-In Drivers

Duffle has a few default drivers:

- `docker`: Runs OCI and Docker images using a local Docker client (currently requires the Docker CLI)
- `aci`: Runs a Docker image inside of Azure ACI (currently requires the `az` command line client)
- `debug`: Dumps the info that was sent to the driver, and exits
- `???`: Runs a VM image on... (TODO: We want a VM version if possible. Maybe `az` for this?)

## Driver-Specific Configuration

Configuration is sent to drivers via environment variables. For example, setting the environment variable `$DEBUG` will turn on debugging for most drivers, while setting the environment variable `$AZ_RESOURCE_GROUP` will set the resource group setting on drivers that use `az`.

## Custom Drivers

Custom drivers are implemented following the pattern of `git` plugins: When a driver is requested (`-d mydriver`) and Duffle does not have built-in support for that driver name, it will seek `$PATH` for a program named `duffle-mydriver` (prepending `duffle-` to the driver name).

If a suitable executable is found, Duffle will execute that program, using the action requested. The environment in which that command executes will be pre-populated with the current environment variables. Credential sets will be passed as well. And the operation will be sent as a JSON body on STDIN:

```json
{
  "Installation": "foo",
  "Action": "install",
  "Parameters": {
      "backend_port": 80,
      "hostname": "localhost"
  },  
  "Credentials": [
      {
          "type": "env",
          "name": "SERVICE_TOKEN",
          "value": "secret"
      }
  ],
  "Image": "bar:1.2.3",
  "ImageType": "docker",
  "Revision": "aaaaaa1234567890"
}
```

The custom driver is expected to take that information and execute the appropriate action for the given image.

### Required Flags for a Driver

A driver must implement two flags:

- `--handles`: Must return a comma-separated list of image types that it can handle
- `--help`: Must return user-friendly documentation on the driver

Only one of these two flags will be provided. No other flags will be sent. When these flags are
sent, no data will be sent over STDIN.

### Example

The following is a simple Bash script that implements both of the required flags, and responds to a request by printing the action and then exiting.

```bash
#!/bin/bash
set -eo pipefail

if [[ $1 == "--handles" ]]; then
    echo docker,oci,qcow
    exit 0;
elif [[ $1 == "--help" ]]; then
    echo "Put yer helptext here"
    exit 1;
fi

echo -n "Plugin: The action is "
cat - | jq .action
```

If the `--handles` flag is set, this will return `docker,oci,qcow` and exit with code 0 (no error). If `--help` is set, the help text will be sent.

Under all other cases, it will attempt to read STDIN, pipe that through the `jq` command, and print the `action` found in the JSON body.

By naming this file `duffle-foo` and placing it in the `$PATH`, we can execute it as a driver:

```console
$ duffle -d foo install myname technosophos/helloworld:0.1.0
Plugin: The action is "install"
```

Note that when it comes to execution order, it will be invoked as follows:

- When Duffle loads, it will look for an internal driver named `foo`.
  - If Duffle finds an internal driver named `foo` (which it won't), it will execute the internal version
  - If Duffle does not find an internal driver named `foo`, it will create a stub command executor for `duffle-foo`.
- When Duffle determines what image type the `bundle.json`, it will run `duffle-foo --handles`.
  - If the declared image type is not in the returned list, Duffle will return an error and quit.
- When the operation is ready, Duffle will run `duffle-foo` and pipe the JSON data into `duffle-foo`'s standard input.
  - if `duffle-foo` returns with an exit code > 1, Duffle will generate an error and exit
  - if `duffle-foo` returns an exit code 0, Duffle will mark this as a successful operation

### Parameters and Credentials for Custom Drivers

The parameters and credentials that are sent to a custom driver will have already been verified.

Parameters will contain the validated, merged parameters. They will be validated against the parameters specification contained in the `bundle.json` file.

Credentials will be loaded and converted to their `destination` format.

A driver MAY withhold some credentials from the underlying system it represents, but it MUST inform the user if doing so.

A driver MUST NOT remove any of the parameters, and must inject them into the image in the format specified by the CNAB specification.

Next Section: [duffle build](203-duffle-build.md)