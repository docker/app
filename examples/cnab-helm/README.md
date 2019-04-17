## Using a CNAB with Docker Application

### Requirements

* [Docker Desktop](https://www.docker.com/products/docker-desktop) with Kubernetes enabled or any other Kubernetes cluster
* Source code from this directory
* [Helm](https://helm.sh) configured for your Kubernetes cluster
* A `duffle` [credential set](https://github.com/deislabs/duffle/blob/1847f2f7127f13f62c1377f936cba522e8947dfb/docs/proposal/201-credentialset.md) created

### Examples

Install the Helm chart example using `docker app`

**Note**: This example comes from
[deislabs/example-bundles](https://github.com/deislabs/example-bundles/tree/d1d95e25a2092ac170d9accd749dffa8babb2e05/hellohelm). See the [license file](./LICENSE) in this directory.

```console
$ docker app install --credential-set mycreds.yml bundle.json
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
$ docker app status --credential-set mycreds.yml hellohelm
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
docker app uninstall --credential-set mycreds.yml hellohelm
Do Uninstall
helm delete --purge hellohelm
release "hellohelm" deleted
```
