package context

import (
	"github.com/docker/cli/cli/context/store"
)

const (
	hostKey          = "host"
	skipTLSVerifyKey = "skipTLSVerify"
	caKey            = "ca.pem"
	certKey          = "cert.pem"
	keyKey           = "key.pem"
)

// EndpointMetaBase contains fields we expect to be common for most context endpoints
type EndpointMetaBase struct {
	ContextName   string
	Host          string
	SkipTLSVerify bool
}

// ToStoreMeta converts the endpoint to the store format
func (e *EndpointMetaBase) ToStoreMeta() store.Metadata {
	return store.Metadata{
		hostKey:          e.Host,
		skipTLSVerifyKey: e.SkipTLSVerify,
	}
}

// EndpointFromContext extracts a context endpoint metadata into a typed EndpointMetaBase structure
func EndpointFromContext(contextName, endpointName string, metadata store.ContextMetadata) *EndpointMetaBase {
	ep, ok := metadata.Endpoints[endpointName]
	if !ok {
		return nil
	}
	host, _ := ep.GetString(hostKey)
	skipTLSVerify, _ := ep.GetBoolean(skipTLSVerifyKey)
	return &EndpointMetaBase{
		ContextName:   contextName,
		Host:          host,
		SkipTLSVerify: skipTLSVerify,
	}
}
