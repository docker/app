# Duffle: The CNAB Package Manager

Duffle is a reference implementation of a package manager and build tool for CNAB bundles. This document reflects the current design thinking for Duffle.

## The Scope of Duffle

Duffle is intended to perform the following tasks:

- Build Duffle images from resources
- Push and pull CNAB bundles to image registries
- Install, upgrade, and delete CNAB images
- Import and export CNAB bundles (e.g. for moving CNABs offline)
- Managing the duffle environment

## Building packages with `duffle build`

the `duffle build` command takes a file-based representation of a CNAB bundle, combined with Duffle configuration, and builds a CNAB bundle.

It builds invocation images. When a Duffle application is multi-container, it also will build those images.

The behavior of `duffle build` is described in [Duffle Build](203-duffle-build.md).

## Duffle Install, Upgrade, Uninstall, and Status

The following commands run a CNAB's applicable invocation image, each passing a different `CNAB_ACTION`:

- `duffle install` captures the intent to install a bundle, and sets `CNAB_ACTION=install`.
- `duffle upgrade` captures the intent to upgrade an existing installation with a bundle, and sets `CNAB_ACTION=upgrade`.
- `duffle uninstall` captures the intent to uninstall an application, and sets `CNAB_ACTION=uninstall`.
- `duffle status` captures the intent to determine the status of an application, and sets `CNAB_ACTION=status`.

All of these actions accept [credential sets](201-credentialsets.md). `install` and `upgrade` accept parameters. (TODO: In the future, `uninstall` may also accept parameters).

Install creates a [claim](104-claims.md).

Upgrade takes a claim's installation name as input, loads the claim, executes the update, and then saves the updated claim.

Uninstall takes a claim's installation name, looks up the claim, executes the uninstall, and then destroys the claim.

Status takes a claim's installation name, looks up a claim, and executes the status. Status never modifies the claim.

## Duffle List

The `duffle list` command lists all claims. It serves as a record of installed (but not uninstalled) applications.

## Exporting a CNAB bundle with `duffle export`

The `duffle export` command exports a invocation image together with all of its associated images, generating a single gzipped tar file as output.

The "thick" representation of an export includes all of the layers of all of the images.

The "thin" representation of an export includes only the invocation image.

## Importing a CNAB bundle with `duffle import`

The `duffle import` command imports an exported Duffle image.

For thick images, it will load the images into the local registry.

For thin images, it will (if necessary) pull the dependent images from a registry and load them into the local Docker/Moby daemon.

## Passing Parameters into the Invocation Image

CNAB includes a method for declaring user-facing parameters that can be changed during certain operations (like installation). Parameters are specified in the `bundle.json` file. Duffle processes these as follows:

- The user may specify values when invoking `duffle install` or similar commands.
- Prior to executing the image, Duffle will read the manifest file, extract the parameter definitions, and then merge specified values and defaults into one set of finalized parameters
- During startup of the image, Duffle will inject each parameter as an environment variable, following the conversion method determined by CNAB:
  - The variable name will be: `CNAB_P_` plus the uppercased variable name (e.g. `CNAB_P_SERVER_PORT`), and the value will be a string representation of the value.

## The Invocation Image Lifecycle

For operations that execute this installation image (install, upgrade, etc.), this is the lifecycle:

- Load the parameters and credential set definitions
- Load the claim (if necessary)
- Load the `bundle.json`. If signed, verify it before parsing the body
- Locate the invocation image
- Validate the parameters and credentials
- Prepare the invocation image, mounting the parameters and credentials, as well as the claim data
  - At this stage, the image type will be matched to a driver
  - If `--driver` is specified, this phase may involve locating the driver
- Execute the invocation image
- Update the claim
- Exit

## TODO

The following items remain to be specified:

- How `duffle init` works
- How Duffle interacts with an image registry (logging in, pushing/pulling bundles)
- Whether Duffle will support multi-runtimes in a single image.

| Method | Description |
| --- | --- |
| naive | A CNAB bundle can have only configurational runtime |
| intentional | A CNAB bundle can expose toggle switches for which runtimes (e.g. Mesos vs Kubernetes), and user chooses |
| automatic | A CNAB bundle may expose multiple runtimes, but automatically choose which applies to the current config |

Next section: [credential set](201-credentialset.md)
