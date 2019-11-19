# Example: From Docker App to CNAB

Docker Apps are Docker’s implementation of the industry standard Cloud Native Application Bundle (CNAB). [CNAB](https://cnab.io/) is an industry specification put in place to facilitate the bundling, sharing, installing and managing of cloud-native apps that are not only made up of containers but also from such things as hosted databases, functions, etc.

Docker App is designed to abstract as many CNAB specifics as possible, to provide users with a tool that is easy to use while alleviating the need to bother with the CNAB specification. 

This example will demonstrate that Docker App is actually leveraging CNAB. To learn more about CNAB, you can refer to the [CNAB specification](https://github.com/cnabio/cnab-spec).


## App Definition

The App definition for this example is ready to use and can be found in the [hello.dockerapp](hello.dockerapp) directory in this folder.


## App Image

Now we are going to build an App image from this App definition.

```shell
$ docker app build . -f hello.dockerapp -t myrepo/cnab-example:1.0.0
[+] Building 0.6s (6/6) FINISHED
(...) (Build output)
sha256:ee61121d6bff0266404cc0077599c1ef7130289fec721
```                                                                              

*Note that a `bundle.json` file has been created in the `/Users/username/.docker/app/bundles/docker.io/myrepo/cnab-example/_tags/1.0.0` directory.*

Open the open the `bundle.json` file in your favorite text editor and you'll see this is a [CNAB bundle](https://github.com/cnabio/cnab-spec).

Copy the `bundle.json`file to your working directory, next to the `hello.dockerapp` App definition.

```shell
$ tree
.
├── bundle.json
└── hello.dockerapp
    ├── docker-compose.yml
    ├── metadata.yml
    └── parameters.yml
```

## Running App

### Run the App from an App image

You can run this App using the `docker app run`command.

```shell
$ docker app run myrepo/cnab-example:1.0.0 --name mycnabexample
Creating network mycnabexample_default
Creating service mycnabexample_hello
App "mycnabexample" running on context "default"
```

Get the list of running Apps using the `docker app ls` command.

```shell
$ docker app ls
RUNNING APP       APP NAME        LAST ACTION  RESULT   CREATED         MODIFIED          REFERENCE
mycnabexample     hello (0.2.0)   install      success  15 minutes ago  15 minutes ago    docker.io/myrepo/cnab-example:1.0.0
```

Then remove the current running App.

```shell
$ docker app rm mycnabexample
Removing service mycnabexample_hello
Removing network mycnabexample_default
```

### Run the App from a CNAB bundle

To demonstrate that Docker App is an implementation of [CNAB](https://cnab.io/), it is also possible to directly run the `bundle.json` file (or any other CNAB bundle) using the `--cnab-bundle-json` experimental flag. 

*Note: To use this flag, you have to enable the experimental mode for the Docker CLI first.*

Open the `/Users/username/.docker/config.json` file in a text editor and change the `"experimental"` field to `"enabled"`.

Run your app passing a `bundle.json` file.

```shell
$ docker app run myrepo/cnab-example:1.0.0 --name mycnabexample --cnab-bundle-json bundle.json
Creating network mycnabexample_default
Creating service mycnabexample_hello
App "mycnabexample" running on context "default"
```

Get the list of running Apps using the `docker app ls` command.

```shell
$ docker app ls
RUNNING APP       APP NAME        LAST ACTION  RESULT   CREATED         MODIFIED          REFERENCE
mycnabexample     hello (0.2.0)   install      success  15 minutes ago  15 minutes ago    docker.io/myrepo/cnab-example:1.0.0
```

Inspect your running app using the `docker app inspect`command.

```shell
$ docker app inspect mycnabexample --pretty
Running App:
  Name: titi
  Created: 1 minute ago
  Modified: 1 minute ago
  Revision: 01DT28SRQZF12FN5YFQ36XCBYS
  Last Action: install
  Result: success
  Ochestrator: swarm

App:
  Name: hello
  Version: 0.2.0

Parameters:
  port: "8765"
  text: Hello!

ID           NAME                  MODE         REPLICAS   IMAGE                  PORTS
c21wxj9ts08y mycnabexample_hello   replicated   1/1        hashicorp/http-echo    *:8765->5678/tcp
```

Finally, remove the current running App.

```shell
$ docker app rm mycnabexample
Removing service mycnabexample_hello
Removing network mycnabexample_default
```