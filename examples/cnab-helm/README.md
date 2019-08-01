## Using a CNAB with Docker Application

### Requirements

* [Docker Desktop](https://www.docker.com/products/docker-desktop) with Kubernetes enabled or any other Kubernetes cluster
* Source code from this directory
* [Helm](https://helm.sh) configured for your Kubernetes cluster
* A [credential set](https://github.com/deislabs/cnab-spec/blob/master/802-credential-sets.md) created

### Examples

Install the Helm chart example using `docker app`:

```console
$ docker app install --credential-set creds.yaml bundle.json
Do install for hellohelm
helm install --namespace hellohelm -n hellohelm /cnab/app/charts/alpine
NAME:   hellohelm
LAST DEPLOYED: Tue Jun 11 15:31:10 2019
NAMESPACE: hellohelm
STATUS: DEPLOYED

RESOURCES:
==> v1/Pod
NAME              READY  STATUS             RESTARTS  AGE
hellohelm-alpine  0/1    ContainerCreating  0         0s


Application "hellohelm" installed on context "default"
```

**Note**: When using Docker Desktop, you will need to change the IP address in
your Kubernetes configuration file from `127.0.0.1` to its internal IP address.

Check the status of the Helm-based application:

```console
$ docker app status --credential-set creds.yaml hellohelm
INSTALLATION
------------
Name:        hellohelm
Created:     39 seconds
Modified:    36 seconds
Revision:    01DD3JM99WRGVAV7T56RMAW13E
Last Action: install
Result:      SUCCESS

APPLICATION
-----------
Name:      hellohelm
Version:   0.1.0
Reference:

PARAMETERS
----------
port: 8080
```

Uninstall the Helm-based application:

```console
docker app uninstall --credential-set creds.yaml hellohelm
Do Uninstall
helm delete --purge hellohelm
release "hellohelm" deleted
Application "hellohelm" uninstalled on context "default"
```

**Note**: This example comes from
[deislabs/example-bundles](https://github.com/deislabs/example-bundles/tree/d1d95e25a2092ac170d9accd749dffa8babb2e05/hellohelm). See the [license file](./LICENSE) in this directory.