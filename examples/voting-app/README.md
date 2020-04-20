# Example: Voting App

In this example, we will create a Docker App from the existing Docker
sample `example-voting-app` from
[here](https://github.com/dockersamples/example-voting-app).

## Creating an App definition

First, create an App definition from an existing [Compose file](https://docs.docker.com/compose/compose-file/) using the `docker app init` command:

```shell
$ docker app init voting-app --compose-file docker-compose.yml
Created "voting-app.dockerapp"
$ tree
.
├── docker-compose.yml
├── voting-app.dockerapp
    ├── docker-compose.yml
    ├── metadata.yml
    └── parameters.yml
```

### Editing metadata

In the `voting-app.dockerapp` directory, open the `metadata.yml` file and fill the "description" and "maintainers" fields.

### Editing services

Now we are going to add some variables to our Compose file. 

To do so, open the `docker-compose.yml` file in the `voting-app.dockerapp` directory, and edit the following values:

* In the `vote` service, change the port from `"5000:80"` to `${vote.port}:80`
* In the `result` service, change the port from `"5001:80"` to `${result.port}:80`
* In the `visualizer` service, change the port from `"8080:8080"` to `${visualizer.port}:8080`
* In the `vote` service, change the replicas from `2` to `${vote.replicas}`

The `voting-app.dockerapp/docker-compose.yml` file should now look like this:

```yaml
version: "3.6"

services:
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
  db:
    image: postgres:9.4
    ports:
      - "5432:5432"
  vote:
    image: dockersamples/examplevotingapp_vote:before
    ports:
      - "${vote.port}:80"
    deploy:
      replicas: ${vote.replicas}
  result:
    image: dockersamples/examplevotingapp_result:before
    ports:
      - "${result.port}:80"
  worker:
    image: dockersamples/examplevotingapp_worker
```

### Set default parameters

Open the `voting-app.dockerapp/parameters.yml` file and define a default value for each variable you created in the `docker-compose.yml`file in the previous step.

```yaml
vote:
  port: 5000
  replicas: 2
result:
  port: 5001
```

## Building an App image

Next, build an App image from the App definition we have created:

```shell
$ docker app build . -f voting-app.dockerapp -t voting-app:0.1.0
[+] Building 0.6s (6/6) FINISHED
(...) (Build output)
sha256:46379a70af728aca32c993373b4a52655fde106953dd5a4e56aed05cde202530
```

## Inspecting an App image

Now let's get detailed information about the App image we just built using the `docker app image inspect` command. Note that the `--pretty` option allows to get a human friendly output rather than the default JSON output.

```shell
$ docker  app image inspect voting-app:0.1.0 --pretty
name: voting-app
description: Dogs or cats?
maintainers:
- name: user
  email: user@email.com


SERVICE REPLICAS PORTS IMAGE
db      1        5432  docker.io/library/postgres:9.4@sha256:c2561ced3d8b82a306fe09b18f9948e2d2ce8b47600125d2c7895ca3ea3a9a44
redis   1        6379  docker.io/library/redis:alpine@sha256:27e139dd0476133961d36e5abdbbb9edf9f596f80cc2f9c2e8f37b20b91d610d
result  1        5001  docker.io/dockersamples/examplevotingapp_result:before@sha256:83b568996e930c292a6ae5187fda84dd6568a19d97cdb933720be15c757b7463
vote    2        5000  docker.io/dockersamples/examplevotingapp_vote:before@sha256:8e64b18b2c87de902f2b72321c89b4af4e2b942d76d0b772532ff27ec4c6ebf6
worker  1              docker.io/dockersamples/examplevotingapp_worker:latest@sha256:55753a7b7872d3e2eb47f146c53899c41dcbe259d54e24b3da730b9acbff50a1

PARAMETER     VALUE
result.port   5001
vote.port     5000
vote.replicas 2
```

Service images inside of a Docker App image are immutable, meaning that the App version ties to a fixed list of service images, and you can see it here: check the service image information in the `docker app image inspect`output above; you can see that each service (`db`, `redis`, `result`, `vote` and `worker`) has a unique service image associated at build time.

*Notes:* 
* *the service image unicity is guaranteed by the tag using a digest (sha256 value)*
* *the "." in the Parameter section indicates hierarchy*

## Running the App

Now, run the App using the `docker app run`command.

```shell
$ docker app run voting-app:0.1.0 --name myvotingapp
Creating network myvotingapp_default
Creating service myvotingapp_vote
Creating service myvotingapp_result
Creating service myvotingapp_worker
Creating service myvotingapp_redis
Creating service myvotingapp_db
App "myvotingapp" running on context "default"
```

You can get detailed information about the running App using the `docker app inspect` command.

```shell
docker app inspect myvotingapp --pretty
Running App:
  Name: myvotingapp
  Created: 43 seconds ago
  Modified: 33 seconds ago
  Revision: 01DT6PJ43CCNWEH4XRMPGSX82A
  Last Action: install
  Result: success
  Ochestrator: swarm

App:
  Name: voting-app
  Version: 0.1.0
  Image Reference: docker.io/library/voting-app:0.1.0

Parameters:
  result.port: "5001"
  vote.port: "5000"
  vote.replicas: "2"

ID           NAME               MODE       REPLICAS IMAGE                                 PORTS
brin0j269w9z myvotingapp_redis  replicated 1/1      redis                                 *:6379->6379/tcp
fdzie3g4712m myvotingapp_worker replicated 1/1      dockersamples/examplevotingapp_worker
mb37mavvj55r myvotingapp_result replicated 1/1      dockersamples/examplevotingapp_result *:5001->80/tcp
vk26ecrvycs8 myvotingapp_db     replicated 1/1      postgres                              *:5432->5432/tcp
yt011xo3yc81 myvotingapp_vote   replicated 2/2      dockersamples/examplevotingapp_vote   *:5000->80/tcp
```

## Adding parameters file for another environment

Create a `prod` folder to store the parameters you would use for
production. Create a new `prod-parameters.yml` file in the `prod` folder.

```shell
$ mkdir prod 
$ tree
.
├── docker-compose.yml
├── prod
│   └── prod-parameters.yml
└── voting-app.dockerapp
    ├── docker-compose.yml
    ├── metadata.yml
    └── parameters.yml
```

Open it in a text editor and set the values you would like to use for production, for example:

```yaml
vote:
  port: 8080
  replicas: 5
result:
  port: 9000
```

## Update the running App using production parameters

Now, we will update the running App to overwrite the current parameters with the production parameters we created at the previous step.

```shell
$ docker app update myvotingapp --parameters-file prod/prod-parameters.yml
Updating service myvotingapp_vote (id: os3s3g4pkmqwd3s1nnk9cmeq7)
Updating service myvotingapp_result (id: y4y4m60imchx0pm7vlehnip8s)
Updating service myvotingapp_worker (id: ergdynkn9u03pz1xe461me1yq)
Updating service myvotingapp_redis (id: fimso41ha11xkqqj19j1ev13o)
Updating service myvotingapp_db (id: ub3vxjiwo1zxc75vzj5mu2vqm)
```

Run again the `docker app inspect` command, check the parameter section in the output and you'll see the parameter values have changed.

```shell
docker app inspect myvotingapp --pretty
Running App:
  Name: myvotingapp
  Created: 15 minutes ago
  Modified: 1 minute ago
  Revision: 01DT6QN7D2R3VKM2QPAQCZ3R1F
  Last Action: upgrade
  Result: success
  Ochestrator: swarm

App:
  Name: voting-app
  Version: 0.1.0
  Image Reference: docker.io/library/voting-app:0.1.0

Parameters:
  result.port: "9000"
  vote.port: "8080"
  vote.replicas: "5"

ID           NAME               MODE       REPLICAS IMAGE                                 PORTS
ergdynkn9u03 myvotingapp_worker replicated 1/1      dockersamples/examplevotingapp_worker
fimso41ha11x myvotingapp_redis  replicated 1/1      redis                                 *:6379->6379/tcp
os3s3g4pkmqw myvotingapp_vote   replicated 5/5      dockersamples/examplevotingapp_vote   *:8080->80/tcp
ub3vxjiwo1zx myvotingapp_db     replicated 1/1      postgres                              *:5432->5432/tcp
y4y4m60imchx myvotingapp_result replicated 1/1      dockersamples/examplevotingapp_result *:9000->80/tcp
```

Finally, remove the current running App using the `docker app rm`command.

```shell
$ docker app rm myvotingapp
Removing service myvotingapp_db
Removing service myvotingapp_redis
Removing service myvotingapp_result
Removing service myvotingapp_vote
Removing service myvotingapp_worker
Removing network myvotingapp_default
```
