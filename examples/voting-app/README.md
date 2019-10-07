## Docker Voting Application

### Initialize project

In this example, we will create a Docker Application from the existing Docker
sample `example-voting-app` from
[here](https://github.com/dockersamples/example-voting-app).

Initialize the application:

```console
$ docker app init voting-app --compose-file docker-compose.yml
```

### Edit metadata

Go to `voting-app.dockerapp/` and open `metadata.yml` and fill the following fields:
- description
- maintainers

### Edit the services

Open `voting-app/docker-compose.yml` and add some variables. Change the:

* `vote` service port from `"5000:80"` to `${vote.port}:80`
* `result` service port from `"5001:80"` to `${result.port}:80`
* `visualizer` service port from `"8080:8080"` to `${visualizer.port}:8080`
* `vote` service replicas from `2` to `${vote.replicas}`

In your `voting-app.dockerapp/docker-compose.yml` you should now have:

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

Open `voting-app.dockerapp/parameters.yml` and list the variables you created
above with a default value. Note that the `.` indicates hierarchy.

```yaml
vote:
  port: 5000
  replicas: 2
result:
  port: 5001
```

### Add a parameters file for an environment

Create a `parameters/` folder to store the parameters you would use for
production.

```console
$ mkdir parameters
```

Create a new file in the `parameters/` folder called `my-environment.yml` and
open it in a text editor. Set the parameters that you would like to use for this
environment, for example:

```yaml
vote:
  port: 8080
  replicas: 5
result:
  port: 9000

```

You can then inspect your application using the parameters specified in this
file as follows:

```yaml
$ docker app inspect voting-app.dockerapp --parameters-file parameters/my-environment.yml
voting-app 0.1.0

Maintained by: user <user@email.com>

Dogs or cats?

Services (5) Replicas Ports Image
------------ -------- ----- -----
worker       1              dockersamples/examplevotingapp_worker
redis        1        6379  redis:alpine
db           1        5432  postgres:9.4
vote         5        8080  dockersamples/examplevotingapp_vote:before
result       1        9000  dockersamples/examplevotingapp_result:before

Parameters (3) Value
-------------- -----
result.port    9000
vote.port      8080
vote.replicas  5
```

**Note**: You can use a parameters file for `install` and `upgrade`
as well.
