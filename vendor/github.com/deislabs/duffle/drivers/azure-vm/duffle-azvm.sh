#!/bin/bash
############
# This is alpha-grade code.
#
# Basically, we implement the top layer of a driver here. The rest of the driver
# is implemented in cnab-azure-vm.py
#
# Note that STDIN gets passed to python, which injects it into the script.
############
pydir=$GOPATH/src/github.com/deislabs/duffle/drivers/azure-vm

if [[ $1 == "--handles" ]]; then
  echo -n "azure-image"
  exit 0
fi
if [[ $1 == "--help" ]]; then
  echo "Help text goes here"
  exit 0
fi

echo "[=== This will take several minutes ===]"
python3 $pydir/cnab-azure-vm.py
