# 20180406 meeting

## Template engine

Problem: We will eventually need ifs, but go-template like solutions break yaml, which means no dependency merging will be possible.
Solution: Add an `enable` optional property on all nodes.

Needs support for '.' in variable names for scoping (current workaround _, but that should be easy to patch in cli)
Needs support for variable in yaml keys.
Needs support for basic arithmetic (at least '!')

## Ops concern

Short term plan is to use compose file override. SF suggest adding a specific format that would restrict what ops can do so they don't break the app with arbitrary compose stuff.

## Storage in registry

New platform type 'app', store the app tarball as a layer.

## Auto populated settings

Make some basic settings available in the 'system' namespace: app metadata, current orchestrator.
