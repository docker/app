package main

import (
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/relocated"
	"github.com/docker/cnab-to-oci/relocation"
)

const (
	// bundlePath is where the CNAB runtime will put the actual Bundle definition
	bundlePath = "/cnab/bundle.json"
	// relocationMapPath is where the CNAB runtime will put the relocation map
	// See https://github.com/cnabio/cnab-spec/blob/master/103-bundle-runtime.md#image-relocation
	relocationMapPath = "/cnab/app/relocation-mapping.json"
)

func getBundle() (*bundle.Bundle, error) {
	return relocated.BundleJSON(bundlePath)
}

func getRelocationMap() (relocation.ImageRelocationMap, error) {
	return relocated.RelocationMapJSON(relocationMapPath)
}

func getRelocatedBundle() (*relocated.Bundle, error) {
	bndl, err := getBundle()
	if err != nil {
		return nil, err
	}

	relocationMap, err := getRelocationMap()
	if err != nil {
		return nil, err
	}

	return &relocated.Bundle{
		Bundle:        bndl,
		RelocationMap: relocationMap,
	}, nil
}
