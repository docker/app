package main

import (
	"github.com/cnabio/cnab-go/bundle"
	"github.com/docker/app/internal/image"
	"github.com/docker/cnab-to-oci/relocation"
)

const (
	// bundlePath is where the CNAB runtime will put the actual AppImage definition
	bundlePath = "/cnab/bundle.json"
	// relocationMapPath is where the CNAB runtime will put the relocation map
	// See https://github.com/cnabio/cnab-spec/blob/master/103-bundle-runtime.md#image-relocation
	relocationMapPath = "/cnab/app/relocation-mapping.json"
)

func getBundle() (*bundle.Bundle, error) {
	return image.BundleJSON(bundlePath)
}

func getRelocationMap() (relocation.ImageRelocationMap, error) {
	return image.RelocationMapJSON(relocationMapPath)
}

func getRelocatedBundle() (*image.AppImage, error) {
	bndl, err := getBundle()
	if err != nil {
		return nil, err
	}

	relocationMap, err := getRelocationMap()
	if err != nil {
		return nil, err
	}

	return &image.AppImage{
		Bundle:        bndl,
		RelocationMap: relocationMap,
	}, nil
}
