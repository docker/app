package main

import (
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/relocated"
)

const (
	// bundlePath is where the CNAB runtime will put the actual Bundle definition
	bundlePath = "/cnab/bundle.json"
)

func getBundle() (*bundle.Bundle, error) {
	return relocated.BundleJSON(bundlePath)
}
