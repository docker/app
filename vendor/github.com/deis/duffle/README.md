# Duffle: The CNAB Installer
[![Build Status](https://cnlabs.visualstudio.com/duffle/_apis/build/status/duffle-CI)](https://cnlabs.visualstudio.com/duffle/_build/latest?definitionId=5)

Duffle is a command line tool that allows you to install and manage CNAB bundles. To learn more about about CNAB and duffle, check out the [docs](docs/000-index.md).

## Getting Started

1. Ensure you're running the latest version of Go (1.11+) by running `$ go version`
    ```console
    $ go version
    go version go1.11 darwin/amd64
    ```

2. Clone this repo:
    ```console
    $ cd $GOPATH/src/github.com/deis/
    $ git clone git@github.com:deis/duffle.git
    $ cd duffle
    ```

3. Vendor dependencies and build the `duffle` binary:
    ```console
    $ make bootstrap build
    ```

4. Run the command to set duffle up on your machine:
    ```console
    $ duffle init
    The following new directories will be created:
    /Users/janedoe/.duffle
    /Users/janedoe/.duffle/logs
    /Users/janedoe/.duffle/plugins
    /Users/janedoe/.duffle/repositories
    /Users/janedoe/.duffle/claims
    /Users/janedoe/.duffle/credentials
    ==> Installing default repositories...
    ==> repo added in 1.096263107s
    ```

5. Build and install your first bundle:

    ```console
    $ duffle build ./examples/helloworld/
    Duffle Build Started: 'helloworld': 01CS02FNS3FTM9907V83GAQQMT
    helloworld: Building CNAB components: SUCCESS ⚓  (1.0090s)

    $ duffle credentials generate helloworld-creds -f examples/helloworld/cnab/bundle.json
    name: helloworld-creds
    credentials:
    - name: quux
      source:
        value: EMPTY
      destination:
        path: pquux

    $ duffle install helloworld-demo -c helloworld-creds -f examples/helloworld/cnab/bundle.json
    Executing install action...

    Install action
    Action install complete for helloworld-demo
    ```

    *Notes:*
    * Learn more about what a bundle is and its components [here](https://github.com/deislabs/cnab-spec/blob/master/100-CNAB.md).
    * Get a feel for what CNAB bundles look like by referencing the [bundles repo](https://github.com/deis/bundles) on github.

# Debugging using VS Code

For instructions on using VS Code to debug the Duffle binary, see [the debugging document](docs/001-debugging.md).
