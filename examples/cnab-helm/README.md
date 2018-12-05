## Requirements

* Working Docker Desktop install with Kubernetes enabled
* `docker-app` installed
* Source code from this directory
* Create a context with `docker-app context`
* Set the `DOCKER_TARGET_CONTEXT` environment variable
* Helm configured for your Kubernetes cluster
* A `duffle` credential set created

## Examples


Install the Helm chart example using `docker-app`


```
$ docker-app install -c local bundle.json
Do install for hellohelm
helm install --namespace hellohelm -n hellohelm /cnab/app/charts/alpine
NAME:   hellohelm
LAST DEPLOYED: Wed Nov 28 13:58:22 2018
NAMESPACE: hellohelm
STATUS: DEPLOYED

RESOURCES:
==> v1/Pod
NAME              AGE
hellohelm-alpine  0s
```


Check the status of the Helm-based application:

```console
$ docker-app status -c local hellohelm
Do Status
helm status hellohelm
LAST DEPLOYED: Wed Nov 28 13:58:22 2018
NAMESPACE: hellohelm
STATUS: DEPLOYED

RESOURCES:
==> v1/Pod
NAME              AGE
hellohelm-alpine  2m
```

Uninstall the Helm-based application:

```console
docker-app uninstall -c local hellohelm
Do Uninstall
helm delete --purge hellohelm
release "hellohelm" deleted
```
