# Duffle Build

This document describes how `duffle build` works, and how it uses the duffle build file: `duffle.json, duffle.yaml, duffle.toml`.

`duffle build` take a path to a directory that contains a duffle build file (`duffle.json, duffle.toml, or duffle.yaml`) to build a Cloud Native Application Bundle (CNAB). In the process, it also builds all of the invocation images specified in the duffle build file.
