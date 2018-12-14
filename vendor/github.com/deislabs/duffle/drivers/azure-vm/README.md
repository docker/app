# Azure VM Driver (`azvm`)

The `duffle-azvm` script provides an Azure VM-based driver for installing CNAB bundles inside of an Azure VM.

Currently this only works on UNIXy operating systems.

## Requirements

- `az` must be installed and configured
- You must `az login`
- You must have access to VMs that have a CNAB bundle inside of them
- You must have python3 and pip3 installed

## Env Vars

- none

## Image Management and Bundle.json

Your CNAB `bundle.json` should define an at least one invocation image like this:

```json
{
    "name": "helloworld",
    "version": "0.1.0",
    "parameters": {},
    "invocationImages": [
        {
            "imageType": "azure-image",
            "image": "duffle-dev/duffle-vm-example-0.1.1"
        }
    ]
}
```

Note that `imageType` is `azure-image` (indicating that it must be looked up in azure images). And `image` is of the form `RESOURCE_GROUP/IMAGE_NAME`.

## Usage

1. Run `make build-drivers`
2. Add `$GOPATH/src/github.com/deislabs/duffle/bin` to your path
3. On the Duffle commands, set the driver to `azvm`

```console
$ duffle install -d azvm foo -f ./examples/helloazure/cnab/bundle.json
az vm create --resource-group duffle-dev --name foo --image duffle-vm-example-0.1.1 --admin-username duff--generate-ssh-keys
{
  "fqdns": "",
  "id": "/subscriptions/SUBID/resourceGroups/duffle-dev/providers/Microsoft.Compute/virtualMachines/foo",
  "location": "westcentralus",
  "macAddress": "00-0D-3A-F8-3D-06",
  "powerState": "VM running",
  "privateIpAddress": "10.0.0.6",
  "publicIpAddress": "13.77.202.157",
  "resourceGroup": "duffle-dev",
  "zones": ""
}
Install action
Action install complete for foo
{
  "additionalProperties": {},
  "endTime": "2018-08-22T23:11:43.844916+00:00",
  "error": null,
  "name": "d796ff40-4c20-4d80-913a-dbb834dd551e",
  "startTime": "2018-08-22T23:11:03.020793+00:00",
  "status": "Succeeded"
}
```

## Building images with Packer

To build images, install Packer, then edit `drivers/azure-vm/azure-packer.json`. You may want to create a `user.json` file and add all your credentials there. For more information on how to set up your Azure account to build images, read [the docs](https://www.packer.io/docs/builders/azure.html), and make sure you [set up your client correctly](https://www.packer.io/docs/builders/azure-setup.html).

From there, use something like this to build:

```console
$ packer build -var-file keys.json azure-packer.json 
```
