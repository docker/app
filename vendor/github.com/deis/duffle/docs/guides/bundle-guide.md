# Build Your First Bundle!

A `bundle` is a CNAB package. In its slimmest form, a bundle contains metadata (in a `bundle.json` file) which points to a image (we call that the `invocation image`) that contains instructions (in a `run` file) on how to install and configure a multi-component cloud native application.

In this guide, you will create a CNAB bundle which does `echo` commands for various actions similar to the [helloworld](https://github.com/deis/duffle/blob/master/examples/helloworld/cnab/app/run) example.

## Create the Directory Structure
```console
$ mkdir -p helloworld/cnab/app
```

### The `app/` directory
The `app/` directory is the where the logic and any supporting files for the invocation image lives. In this directory, create a `run` file.

```console
$ cd helloworld/cnab/app
$ touch run
$ chmod 755 run # make run file an executable
```

The `run` file for this example is a bash script that acts on environment variables that have already been set.
In `cnab/app/run`:
```
#!/bin/sh

#set -eo pipefail

action=$CNAB_ACTION
name=$CNAB_INSTALLATION_NAME
param=${CNAB_P_HELLO}

echo "$param world"
case $action in
    install)
    echo "Install action"
    ;;
    uninstall)
    echo "uninstall action"
    ;;
    upgrade)
    echo "Upgrade action"
    ;;
    downgrade)
    echo "Downgrade action"
    ;;
    status)
    echo "Status action"
    ;;
    *)
    echo "No action for $action"
    ;;
esac
echo "Action $action complete for $name"
```

The `$CNAB_ACTION` environment variable describes what action is to be performed. `$CNAB_INSTALLATION_NAME` is the name of the instance of the installation of the bundle. Any environment variable that has a prefix of `$CNAB_P_` is a parameter that either had a default set or was passed in at runtime by the end user.

## Defining the Invocation Image
Create a Dockerfile for the invocation image which defines the `run` tool.
```console
$ cd helloworld/cnab
$ touch Dockerfile
```

In, `Dockerfile`:
```
FROM alpine:latest

COPY app/run /cnab/app/run
COPY Dockerfile cnab/Dockerfile

```

## A Tale of Two JSON Files
Every bundle needs a `bundle.json` file. This file will live in the root of the bundle. This file contains metadata about the bundle, information on the required parameters necessary credentials for a successful installation, and content digests for each image specified for the bundle installation. You can write this file by hand OR you can write a file called `duffle.json` in the root of your bundle which specifies registry information and use the `duffle build` command to push images and generate a proper `bundle.json` file. In this example, let's use the `duffle build` command.

```console
$ cd helloworld/
$ touch duffle.json
```

In `duffle.json`:
```
{
    "name": "helloworld",
    "components": {
        "cnab": {
            "name": "cnab",
            "builder": "docker",
            "configuration": {
                "registry": "microsoft"
            }
        }
    },
    "images": [],
    "parameters": {
        "hello": {
          "defaultValue": "hello",
          "type": "string"
        }
    },
    "credentials": {
        "quux": {
            "path": "pquux",
            "env": "equux"
        }
    }
}
```

WARNING: Replace the `registry` field in the file above with the name of your own registry.

## Building the Artifacts
In `helloworld/cnab/`:
```console
$ cd .. # Go to root of your bundle
$ pwd
helloworld/
$ duffle build
Duffle Build Started: 'helloworld': 01CT1YHH79CNN66KMC2Y9T1E1D
helloworld: Building CNAB components: SUCCESS ⚓  (1.0000s)
```

`duffle build` builds the invocation image(s) and creates a `bundle.json` file.

Push the invocation image:
```console
# replace `microsoft` with your own registry
$ docker push microsoft/helloworld-cnab:0.1.0"
```

## Watch it Work
```console
$ duffle install helloworld -f helloworld/cnab/bundle.json
hello world
Install action
Action install complete for helloworld
```
The output of `duffle install` comes from the run script. `hello world` is printed before the defined action is executed. In this example, the action being executed is the install action. In this example, the install action is running `echo 'Install Action'` At the end, the run script prints a message indication the action has been completed.

## Notes and Next steps
- There are alternatives to defining a custom `run` tool. See examples of more complex and different bundles [here](https://github.com/deis/bundles).
- Read more about the CNAB spec in the [docs](https://github.com/deislabs/cnab-spec/blob/master/100-CNAB.md)
