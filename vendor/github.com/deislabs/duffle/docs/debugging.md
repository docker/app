# Debugging Duffle using VS Code

We can use VS Code to breakpoint debug the Duffle binary. For this, you need:

- [VS Code](https://code.visualstudio.com/download)
- [the Go extension for VS Code](https://code.visualstudio.com/docs/languages/go)
- [the Delve debugger](https://github.com/derekparker/delve)

> There is a wiki on the VS Code Go extension repository about [debugging Go code using VS Code and Delve](https://github.com/Microsoft/vscode-go/wiki/Debugging-Go-code-using-VS-Code).

There are two modes for debugging Go code in VS Code that are relevant for us:

- `debug` -  rebuilds the binary on every new run
- `exec` - uses an already built binary, launches it and attaches the debugger

For this document, we are going to use the `exec` mode, simply because we already have a customized way of building the Duffle binary (using `make`), and we might want to iterate through the execution of a command multiple times without rebuilding the binary.

Here are the steps involved in doing breakpoint debugging:

- `make debug` - this will build the Duffle binary and include the debugging symbols
- edit `.vscode/launch.json` and in the `args` field add the Duffle command you want to debug, while passing each argument as a new string. 
- add a breakpoint in `cmd/duffle/install`

For example:

 `$ duffle install debug-test --driver debug -f examples/helloworld/cnab/bundle.json` is translated into 

```json
"args": [
    "install", "debug-test", "--driver", "debug", "-f", "${workspaceFolder}/examples/helloworld/cnab/bundle.json"
]
```

Things to note regarding the path:

- because the `examples` directory is at the root of the repo, the `${workspaceFolder}` variable is need to avoid passing the absolute path
- if running on Windows, the path separator must be `\\`, so the above path becomes `${workspaceFolder}\\examples\\helloworld\\cnab\\bundle.json`
- you can find the full example for both Windows and Unix in `.vscode/launch.json`
