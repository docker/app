## Simple wordpress + mysql app

### Visualize app configuration

```yaml
# docker-app render simple
version: "3.6"
services:
  mysql:
    deploy:
      mode: replicated
      replicas: 1
    environment:
      MYSQL_DATABASE: wordpressdata
      MYSQL_PASSWORD: wordpress
      MYSQL_ROOT_PASSWORD: axx[<^cz3d.fPb
      MYSQL_USER: wordpress
    image: mysql:8
    volumes:
    - type: volume
      source: db_data
      target: /var/lib/mysql
  wordpress:
    deploy:
      mode: global
      replicas: 0
    environment:
      WORDPRESS_DB_HOST: mysql
      WORDPRESS_DB_NAME: wordpressdata
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DEBUG: "true"
    image: wordpress
networks: {}
volumes:
  db_data:
    name: db_data
secrets: {}
configs: {}
```

**Merge with override Compose file**. This example replaces cleartext DB passwords to use secrets instead.

```yaml
# docker-app render simple -c with-secrets.yml
version: "3.6"
services:
  mysql:
    deploy:
      mode: replicated
      replicas: 1
    environment:
      MYSQL_DATABASE: wordpressdata
      MYSQL_PASSWORD: ""
      MYSQL_PASSWORD_FILE: /run/secrets/simple_app_userpass
      MYSQL_ROOT_PASSWORD: ""
      MYSQL_ROOT_PASSWORD_FILE: /run/secrets/simple_app_rootpass
      MYSQL_USER: wordpress
    image: mysql:8
    secrets:
    - source: mysql_rootpass
    - source: mysql_userpass
    volumes:
    - type: volume
      source: db_data
      target: /var/lib/mysql
  wordpress:
    deploy:
      mode: global
      replicas: 0
    environment:
      WORDPRESS_DB_HOST: mysql
      WORDPRESS_DB_NAME: wordpressdata
      WORDPRESS_DB_PASSWORD: ""
      WORDPRESS_DB_PASSWORD_FILE: /run/secrets/sumple_app_userpass
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DEBUG: "true"
    image: wordpress
    secrets:
    - source: mysql_userpass
networks: {}
volumes:
  db_data:
    name: db_data
secrets:
  mysql_rootpass:
    name: simple_app_rootpass
    external: true
  mysql_userpass:
    name: simple_app_userpass
    external: true
configs: {}
```

**Override default settings**. This example sets `debug` to false.

```yaml
# docker-app render simple -s prod-settings.yml
version: "3.6"
[...]
    environment:
      WORDPRESS_DB_HOST: mysql
      WORDPRESS_DB_NAME: wordpressdata
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DEBUG: "false"
[...]
```

**Override from the command line**. This example sets `debug` to false and the database user to a
different value.
```yaml
# docker-app render simple -e debug=true -e mysql.user.name=mollydock
version: "3.6"
services:
  mysql:
[...]
    environment:
      MYSQL_DATABASE: wordpressdata
      MYSQL_PASSWORD: wordpress
      MYSQL_ROOT_PASSWORD: axx[<^cz3d.fPb
      MYSQL_USER: mollydock
[...]
  wordpress:
[...]
    environment:
      WORDPRESS_DB_HOST: mysql
      WORDPRESS_DB_NAME: wordpressdata
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_USER: mollydock
      WORDPRESS_DEBUG: "false"
[...]
```

### View app metadata

```yaml
# docker-app inspect simple
version: 0.1.0
name: simple
description: ""
author: sakuya.izayoi
targets:
  swarm: true
  kubernetes: true
```

### Generate helm package

`docker-app helm simple` will output a Helm package in the `./simple.helm` folder. `-c`, `-e` and `-s` flags apply the same way they do for the `render` subcommand.

```
$ docker-app helm simple -c with-secrets.yml -s prod-settings.yml -e mysql.user.name=mollydock
$ tree simple.helm
simple.helm/
├── Chart.yaml
└── templates
    └── stack.yaml

1 directory, 2 files
$ cat simple.helm/templates/stack.yaml
typemeta:
  kind: stacks.compose.docker.com
  apiversion: v1beta2
objectmeta:
[...]
```

### Generate distributable app package

`docker-app save simple` creates a Docker image packaging the relevant configuration files:

```
$ docker-app save simple
$ docker images simple.dockerapp
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
simple.dockerapp   latest              61f8cafb7762        4 minutes ago       1.2kB
```

The package can later be retrieved using `docker-app load`:

```
$ rm -rf simple.dockerapp
$ docker-app load simple.dockerapp
$ tar -tf simple.dockerapp  # TODO: should unpack automatically?
metadata.yml
docker-compose.yml
settings.yml
$ mv simple.dockerapp simple  # TODO: fix UX
$ docker-app unpack simple
$ tree simple.dockerapp
./simple.dockerapp/
├── metadata.yml
├── docker-compose.yml
└── settings.yml

0 directories, 3 files
```

### Archive app package

`docker-app pack simple` creates a tar archive containing the relevant configuration files:

```
$ docker-app pack simple -o myapp.tar
$ tar -tf myapp.tar
metadata.yml
docker-compose.yml
settings.yml
```

