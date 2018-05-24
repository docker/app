## The Docker voting app

### Initialize project

In this example, we will create an app from the existing docker sample `example-voting-app`. First download the project [here](https://github.com/dockersamples/example-voting-app).

Initialize the project using `docker-app init voting-app -c example-voting-app/docker-stack.yml`.

### Edit metadata

Go to `voting-app.dockerapp/` and open `metadata.yml` and fill the following fields:
- description
- maintainers

### Add variables to the compose file

Open `docker-compose.yml` and start by changing the version to `3.2` (generated docker-compose are version 3.2+ compatible). Change constants you want by variables, e.g.:

Change the images used, from:
- `dockersamples/examplevotingapp_vote:before` to `${vote.image.name}:${vote.image.tag}`
- `dockersamples/examplevotingapp_result:before` to `${result.image.name}:${result.image.tag}`
- `dockersamples/examplevotingapp_worker:before` to `${worker.image.name}:${worker.image.tag}`
- `dockersamples/examplevotingapp_visualizer:before` to `${visualizer.image.name}:${visualizer.image.tag}`

Change exposed ports, from:
- `<value:5000>` to `${vote.port}`
- `<value:5001>` to `${result.port}`
- `<value:8080>` to `${visualizer.port}`

Change default replicas, from:
- `<value:2>` to `${vote.replicas}`
- `<value:1>` to `${result.replicas}`
- `<value:1>` to `${worker.replicas}`

### Give variables their default value

Open `settings.yml` and add every variables with the default value you want, e.g.:

```
$ cat settings.yml
# Vote.
vote:
  image:
    name: dockersamples/examplevotingapp_vote
    tag: latest
  port: 8080
  replicas: 1

# Result.
result:
  image:
    name: dockersamples/examplevotingapp_result
    tag: latest
  port: 8181
  replicas: 1

# Visualizer.
visualizer:
  image:
    name: dockersamples/visualizer
    tag: latest
  port: 8282

# Worker.
worker:
  image:
    name: dockersamples/examplevotingapp_worker
    tag: latest
  replicas: 1
```

Test your application by running `docker-app render`.

### Add settings for production and development environments

Create `settings/development.yml` and `settings/production.yml` and add your target-specific variables.

```
$ cat settings/development.yml
# Vote.
vote:
  image:
    name: vote

# Result.
result:
  image:
    name: result
```
```
$ cat settings/production.yml
# Vote.
vote:
  port: 80
  replicas: 3

# Result.
result:
  port: 80
  replicas: 5
```

### Wrap everything in a Makefile

Add a Makefile to simplify rendering, deploying and killing your app.

```
$ cat Makefile
# Input.
APP_NAME := voting-app
SETTINGS_DIR ?= settings

# Output.
DEVELOPMENT_DIR := build/development
PRODUCTION_DIR := build/production

#
# Cleanup.
#
cleanup/production:
	@rm -rf $(PRODUCTION_DIR)

cleanup/development:
	@rm -rf $(DEVELOPMENT_DIR)

cleanup: cleanup/production cleanup/development

#
# Render.
#
render/production: cleanup/production
	@mkdir -p $(PRODUCTION_DIR)
	docker-app render -f $(SETTINGS_DIR)/production.yml > $(PRODUCTION_DIR)/docker-compose.yml

render/development: cleanup/development
	@mkdir -p $(DEVELOPMENT_DIR)
	docker-app render -f $(SETTINGS_DIR)/development.yml > $(DEVELOPMENT_DIR)/docker-compose.yml

render: render/production render/development

#
# Kill.
#
kill/production:
	docker stack rm ${APP_NAME}

kill/development:
	docker stack rm ${APP_NAME}-dev

kill: kill/production kill/development

#
# Deploy.
#
deploy/production: render/production kill/production
	docker-app deploy -f $(SETTINGS_DIR)/production.yml

deploy/development: render/development kill/development
	docker-app deploy -f $(SETTINGS_DIR)/development.yml

#
# Pack.
#
pack:
	docker-app pack -o $(PACK)

#
# Save.
#
save:
    docker-app save -p

#
# Helm.
#
helm/production:
	docker-app helm -f $(SETTINGS_DIR)/production.yml

helm/development:
	docker-app helm -f $(SETTINGS_DIR)/development.yml
```

You can add more commands, depending on your needs.
