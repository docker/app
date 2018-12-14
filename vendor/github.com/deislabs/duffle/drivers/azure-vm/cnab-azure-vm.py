import json
import sys
from fabric import Connection
from invoke import run

#####
# This script is alpha-quality
#
# This script accepts the JSON payload of a Duffle driver, and then runs a VM-based
# CNAB bundle.
#
# The workflow is:
# - Parse JSON STDIN data
# - Create a new VM
# - Parse the results of the VM creation, and use that to SSH to the new VM
# - Execute the CNAB /cnab/app/run command, holding the connection open
# - Report success/failure
# - Destroy the VM

config = json.load(sys.stdin)

name = config["installation_name"]
bundleRef = config["image"].split("/", 2)
group = bundleRef[0]
image = bundleRef[1]
admin = "duff"
action = config["action"]
params = config["parameters"]
rev = config["revision"]
# credentials = config["credentials"]

# Create a new AZ VM
create = "az vm create --resource-group {} --name {} --image {} --admin-username {} --generate-ssh-keys".format(
    group,
    name,
    image,
    admin
)
print(create)
create_vm = run(create)

if create_vm.failed:
    print("Failed to create VM. Exiting.")
    exit(1)

# Parse the JSON results and get the IP address
vm = json.loads(create_vm.stdout)
myIP = vm['publicIpAddress']
c = Connection(host=myIP, user=admin)

# TODO: Find a better way to set env vars in Fabric.
# TODO: Not all of the env vars are set here!
result = c.run('CNAB_ACTION={} CNAB_INSTALLATION_NAME="{}" /cnab/app/run'.format(action, name))

exitCode = 0
if result.failed:
    exitCode = 1

# Tear down the VM
run("az vm delete --yes --name {} --resource-group {}".format(name, group))
exit(exitCode)