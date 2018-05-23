## The Docker voting app

### Initialize project

In this example, we will create a single service application that deploys a web page displaying a message.

Initialize the project using `docker-app init hello-world`.

### Edit metadata

Go to `hello-world.dockerapp/` and open `metadata.yml` and fill the following fields:
- description
- maintainers

### Add variables to the compose file

Open `docker-compose.yml` and add a service `hello`.

```
services:
  hello:
    image: hashicorp/http-echo
    command: ["-text", "${text}"]
    ports:
      - ${port}:5678
```

### Give variables their default value

Open `settings.yml` and add every variables with the default value you want, e.g.:

```
$ cat settings.yml
port: 8080
text: world
```

### Render

You can now render your application by running `docker-app render` or even personalize the rendered compose file by running `docker-app render -s text="hello user"`.

Create a `render` directory and redirect the output of `docker-app render ...` to `render/docker-compose.yml`.

```
mkdir -p render
docker-app render -s text="hello user" > render/docker-compose.yml
```

### Deploy

Deploy your application by running `docker-app deploy -c render/docker-compose.yml`. http://<ip_of_your_node>:8080 will display your message, e.g. http://127.0.0.1:8080 if you deployed locally.
