## The Docker voting app

### Initialize project

In this example, we will create an app from the existing Docker sample `example-voting-app`. First download the project [here](https://github.com/dockersamples/example-voting-app).

Initialize the project using `docker-app init voting-app --compose-file example-voting-app/docker-stack.yml`.

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

---

[voting-app.dockerapp/docker-compose.yml](voting-app.dockerapp/docker-compose.yml):
```yml
[...]
vote:
    image: ${vote.image.name}:${vote.image.tag}
    ports:
      - ${vote.port}:80
    networks:
      - frontend
    depends_on:
      - redis
    deploy:
      replicas: ${vote.replicas}
      update_config:
        parallelism: 2
      restart_policy:
        condition: on-failure

  result:
    image: ${result.image.name}:${result.image.tag}
    ports:
      - ${result.port}:80
    networks:
      - backend
    depends_on:
      - db
    deploy:
      replicas: ${result.replicas}
      update_config:
        parallelism: 2
        delay: 10s
      restart_policy:
        condition: on-failure

  worker:
    image: ${worker.image.name}:${worker.image.tag}
    networks:
      - frontend
      - backend
    deploy:
      mode: replicated
      replicas: ${worker.replicas}
      labels: [APP=VOTING]
      restart_policy:
        condition: on-failure
        delay: 10s
        max_attempts: 3
        window: 120s
      placement:
        constraints: [node.role == manager]

  visualizer:
    image: ${visualizer.image.name}:${visualizer.image.tag}
    ports:
      - ${visualizer.port}:8080
    stop_grace_period: 1m30s
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock"
    deploy:
      placement:
        constraints: [node.role == manager]
[...]
```

### Give variables their default value

Open `settings.yml` and add every variables with the default value you want, e.g.:

---

[voting-app.dockerapp/settings.yml](voting-app.dockerapp/settings.yml):
```yml
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

---

[voting-app.dockerapp/settings/development.yml](voting-app.dockerapp/settings/development.yml):
```yml
# Vote.
vote:
  image:
    name: vote

# Result.
result:
  image:
    name: result
```
---

[voting-app.dockerapp/settings/production.yml](voting-app.dockerapp/settings/production.yml):
```yml
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

---

[voting-app.dockerapp/Makefile](voting-app.dockerapp/Makefile):
```Makefile
# Input.
SETTINGS_DIR ?= settings
APP_NAME := voting-app

# Output.
DEVELOPMENT_DIR := build/development
PRODUCTION_DIR := build/production
PACK := $(APP_NAME).pack

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
	docker-app render --settings-files $(SETTINGS_DIR)/production.yml > $(PRODUCTION_DIR)/docker-compose.yml

render/development: cleanup/development
	@mkdir -p $(DEVELOPMENT_DIR)
	docker-app render --settings-files $(SETTINGS_DIR)/development.yml > $(DEVELOPMENT_DIR)/docker-compose.yml

render: render/production render/development

#
# Stop.
#
stop/production:
	docker stack rm ${APP_NAME}

stop/development:
	docker stack rm ${APP_NAME}-dev

stop: stop/production stop/development

#
# Deploy.
#
deploy/production: render/production stop/production
	docker-app deploy --settings-files $(SETTINGS_DIR)/production.yml

deploy/development: render/development stop/development
	docker-app deploy --settings-files $(SETTINGS_DIR)/development.yml

#
# Pack.
#
pack:
	docker-app pack -o $(PACK)

#
# Helm.
#
helm/production:
	docker-app helm --settings-files $(SETTINGS_DIR)/production.yml

helm/development:
	docker-app helm --settings-files $(SETTINGS_DIR)/development.yml
```

You can add more commands, depending on your needs.
