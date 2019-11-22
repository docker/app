# Example: Dockercoins

In this example, we will create a Docker App where service images are built along with the App image.

The [dockercoins](https://github.com/dockersamples/dockercoins) demo application is made up of five services:

* `rng` is a web service generating random bytes
* `hasher` is a web service computing hash of POSTed data
* `worker` is a background process using rng and hasher
* `webui` is the web interface to watch progress
* `redis` is handling storage

## App Definition

The App definition for this example is ready to use and can be found in the [coins.dockerapp](coins.dockerapp) directory in this folder.

Open the `coins.dockerapp/docker-compose.yml` file in a text editor:

```yaml
version: "3.7"

services:
  rng:
    build: rng
    ports:
      - "${rng.port}:80"

  hasher:
    build: hasher
    ports:
      - "${hasher.port}:80"

  webui:
    build: webui
    ports:
      - "${webui.port}:80"

  redis:
    image: redis

  worker:
    build: worker
```

You can notice that the `rng`, `webui`, `hasher` and `worker` services all have a `build` field, i.e. each service has a Dockerfile describing how the service image must be built. 

```shell
├── coins.dockerapp
│   ├── docker-compose.yml
│   ├── metadata.yml
│   └── parameters.yml
├── hasher
│   ├── Dockerfile
│   └── hasher.rb
├── rng
│   ├── Dockerfile
│   └── rng.py
├── webui
│   ├── Dockerfile
│   ├── files
│   │   ├── d3.min.js
│   │   ├── index.html
│   │   ├── jquery-1.11.3.min.js
│   │   ├── jquery.js -> jquery-1.11.3.min.js
│   │   ├── rickshaw.min.css
│   │   └── rickshaw.min.js
│   └── webui.js
└── worker
    ├── Dockerfile
    └── worker.py
```

## App Image

Now we are going to build an App image from this App definition. At build time, Docker App is going to build each service image then build the App image embedding the service images.

```shell
$ docker app build -f coins.dockerapp -t myrepo/coins:0.1.0 .
[+] Building 10.5s (37/37) FINISHED
 => [rng internal] load build definition from Dockerfile                                                                                              0.0s
 (...) (some build output)
 => [webui internal] load build definition from Dockerfile                                                                                            0.1s
 => [hasher internal] load build definition from Dockerfile                                                                                           0.1s
 => [worker internal] load build definition from Dockerfile                                                                                           0.1s
 (...) (rest of build output)
 sha256:ee61121d6bff0266404cc0077599c1ef7130289fec721
```

If you browse the `docker app build` command output, you will see that:
* the `rng`, `webui`, `hasher` and `worker` service images have been built from a Dockerfile 
* if you don't have it already loacally, the `redis` image will be pulled

## Running App

You can now run this App using the `docker app run` command.

```shell
$ docker app run myrepo/coins:0.1.0
Creating network dreamy_albattani_default
Creating service dreamy_albattani_hasher
Creating service dreamy_albattani_webui
Creating service dreamy_albattani_redis
Creating service dreamy_albattani_worker
Creating service dreamy_albattani_rng
App "dreamy_albattani" running on context "default"
```

*Note: if you don't pass the `--name` flag to the `docker app run` command, a name for the running App will be automatically generated.*

You list the running Apps using the `docker app ls` command.

```shell
$ docker app ls
RUNNING APP        APP NAME       LAST ACTION   RESULT    CREATED              MODIFIED             REFERENCE
dreamy_albattani   coins (0.1.0)  install       success   About a minute ago   About a minute ago   docker.io/myrepo/coins:0.1.0
```

## Inspect the running App

Get detailed information about the running App using the `docker app inspect` command. Note that the `--pretty` option allows to get a human friendly output rather than the default JSON output.

```shell
$ docker app inspect dreamy_albattani --pretty
Running App:
  Name: dreamy_albattani
  Created: 3 minutes ago
  Modified: 3 minutes ago
  Revision: 01DT9CAEJ6TY48YMRKB4EWW357
  Last Action: install
  Result: success

App:
  Name: coins
  Version: 0.1.0
  Image Reference: docker.io/myrepo/coins:0.1.0

Parameters:
  hasher.port: "8002"
  rng.port: "8001"
  webui.port: "8000"

ID           NAME                    MODE       REPLICAS IMAGE                                                                   PORTS
adpmt82ejfrm dreamy_albattani_worker replicated 1/1      sha256:8c016797b7042227d224ce058ed099f3838904a8f8a259d0e000440851c648a1
r5a8ukf2j17a dreamy_albattani_redis  replicated 1/1      redis
sffx1pe1b04u dreamy_albattani_hasher replicated 1/1      sha256:7ab1468f5e2b6ff8ece16b56832fa6b3547bf71375b6d71c55211e2dbe24ba11 *:8002->80/tcp
uk8cixh15pob dreamy_albattani_webui  replicated 1/1      sha256:75279ded158d53fc69ef7570f3e8d5e2646479bb50ddf03b9b06d24d39815ce3 *:8000->80/tcp
ypx32ze6b0wt dreamy_albattani_rng    replicated 1/1      sha256:9bc51dbbbdffb342468289b5bf8ad411fe2d6bdbac044cc69075c33df54919a2 *:8001->80/tcp
```

## Remove running App and App image

Now, let's remove the running App.

```shell
$ docker app rm dreamy_albattani
Removing service dreamy_albattani_hasher
Removing service dreamy_albattani_redis
Removing service dreamy_albattani_rng
Removing service dreamy_albattani_webui
Removing service dreamy_albattani_worker
Removing network dreamy_albattani_default
```

And finally, let's remove the App image.

```shell
$ docker app image rm myrepo/coins:0.1.0
Deleted: myrepo/coins:0.1.0
```