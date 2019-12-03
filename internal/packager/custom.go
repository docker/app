package packager

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
)

const (
	// DockerAppCustomVersion1_0_0 is the custom payload version 1.0.0
	DockerAppPayloadVersion1_0_0 = "1.0.0"

	// DockerAppCustomVersionCurrent the current payload version
	// The version must be bumped each time the Payload format change.
	DockerAppPayloadVersionCurrent = DockerAppPayloadVersion1_0_0
)

// DockerAppCustom contains extension custom data that docker app injects
// in the bundle.
type DockerAppCustom struct {
	// Payload format version
	Version string `json:"version,omitempty"`
	// Custom payload format depends on version
	Payload json.RawMessage `json:"payload,omitempty"`
}

// CustomPayloadAppVersion is a custom payload with a docker app version
type CustomPayloadAppVersion interface {
	AppVersion() string
}

type payloadV1_0 struct {
	Version string `json:"app-version"`
}

func (p payloadV1_0) AppVersion() string {
	return p.Version
}

func newCustomPayload() (json.RawMessage, error) {
	p := payloadV1_0{Version: internal.Version}
	j, err := json.Marshal(&p)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// CheckAppVersion prints a warning if the bundle was built with a different version of docker app
func CheckAppVersion(stderr io.Writer, bndl *bundle.Bundle) error {
	payload, err := CustomPayload(bndl)
	if err != nil {
		return err
	}
	if payload == nil {
		return nil
	}

	var version string
	if versionPayload, ok := payload.(CustomPayloadAppVersion); ok {
		version = versionPayload.AppVersion()
	}
	if version != internal.Version {
		fmt.Fprintf(stderr, "WARNING: App Image has been built with a different version of docker app: %q\n", version)
	}
	return nil
}

// CustomPayload parses and returns the bundle's custom payload
func CustomPayload(b *bundle.Bundle) (interface{}, error) {
	custom, err := parseCustomPayload(b)
	if err != nil {
		return nil, err
	}

	switch version := custom.Version; version {
	case DockerAppPayloadVersion1_0_0:
		var payload payloadV1_0
		if err := json.Unmarshal(custom.Payload, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	default:
		return nil, nil
	}
}

func parseCustomPayload(b *bundle.Bundle) (DockerAppCustom, error) {
	customMap, ok := b.Custom[internal.CustomDockerAppName]
	if !ok {
		return DockerAppCustom{}, nil
	}

	customJSON, err := json.Marshal(customMap)
	if err != nil {
		return DockerAppCustom{}, err
	}

	var custom DockerAppCustom
	if err = json.Unmarshal(customJSON, &custom); err != nil {
		return DockerAppCustom{}, err
	}

	return custom, nil
}
