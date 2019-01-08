## Simple wordpress + mysql app

### Visualize app configuration

```yaml
# docker-app render wordpress
version: "3.6"
services:
  mysql:
    deploy:
      mode: replicated
      replicas: 1
      endpoint_mode: dnsrr
    environment:
      MYSQL_DATABASE: wordpressdata
      MYSQL_PASSWORD: wordpress
      MYSQL_ROOT_PASSWORD: axx[<^cz3d.fPb
      MYSQL_USER: wordpress
    image: mysql:5.6
    networks:
      overlay: null
    volumes:
    - type: volume
      source: db-data
      target: /var/lib/mysql
  wordpress:
    depends_on:
    - mysql
    deploy:
      mode: replicated
      replicas: 1
      endpoint_mode: vip
    environment:
      WORDPRESS_DB_HOST: mysql
      WORDPRESS_DB_NAME: wordpressdata
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DEBUG: "true"
    image: wordpress
    networks:
      overlay: null
    ports:
    - mode: ingress
      target: 80
      published: 8080
      protocol: tcp
networks:
  overlay: {}
volumes:
  db-data:
    name: db-data
```

**Override default parameters with file**. This example sets `debug` to `"false"` and the wordpress service published port to 80 as defined in `prod-parameters.yml`.

```yaml
# docker-app render wordpress --parameters-files prod-parameters.yml
version: "3.6"
[...]
    environment:
      WORDPRESS_DB_HOST: mysql
      WORDPRESS_DB_NAME: wordpressdata
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DEBUG: "false"
[...]
    ports:
    - mode: ingress
      target: 80
      published: 80
      protocol: tcp
[...]
```

**Override from the command line**. This example sets `debug` to `"false"` and the database user to a different value.
```yaml
# docker-app render wordpress --set debug=\"true\" --set mysql.user.name=mollydock
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
      WORDPRESS_DEBUG: "true"
[...]
```

### View app metadata

```yaml
# docker-app inspect wordpress
wordpress 0.1.0
Maintained by: sakuya.izayoi <sizayoi@sdmansion.jp>

Parameters                    Default
----------                    -------
debug                         true
mysql.database                wordpressdata
mysql.image.version           5.6
mysql.rootpass                axx[<^cz3d.fPb
mysql.scale.endpoint_mode     dnsrr
mysql.scale.mode              replicated
mysql.scale.replicas          1
mysql.user.name               wordpress
mysql.user.password           wordpress
volumes.db_data.name          db-data
wordpress.port                8080
wordpress.scale.endpoint_mode vip
wordpress.scale.mode          replicated
wordpress.scale.replicas      1
```

### Generate distributable app package

**Note:** If using Windows, this only works in Linux container mode.

`docker-app save wordpress` creates a Docker image packaging the relevant configuration files:

```
$ docker-app save wordpress
$ docker-app ls
REPOSITORY            TAG                 IMAGE ID            CREATED             SIZE
wordpress.dockerapp   latest              61f8cafb7762        4 minutes ago       1.2kB
```

The image can be pushed to the hub:
```
$ docker push --namespace <username> wordpress
The push refers to repository [docker.io/<username>/wordpress.dockerapp]
61f8cafb7762: Pushed
latest: digest: sha256:91b9b526ac1e645e9c89663ff1453c2d7f68535e2dbbca6d4466d365e15ee155 size: 525
```

One can now deploy the application using `docker-app deploy`:

```
$ docker-app deploy <username>/wordpress
Creating network wordpress_overlay
Creating service wordpress_mysql
Creating service wordpress_wordpress
```
