package packager

import (
	"encoding/json"
	"time"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
)

const (
	// DockerAppCustomVersion1_0_0 is the custom payload version 1.0.0
	DockerAppCustomVersion1_0_0 = "1.0.0"

	// DockerAppCustomVersionCurrent the current payload version
	DockerAppCustomVersionCurrent = DockerAppCustomVersion1_0_0
)

// DockerAppCustom contains extension custom data that docker app injects
// in the bundle.
type DockerAppCustom struct {
	Version string          `json:"version,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// CustomPayloadCreated is a custom payload with a created time
type CustomPayloadCreated interface {
	CreatedTime() time.Time
}

type payloadV1_0 struct {
	Created time.Time `json:"created"`
}

func (p payloadV1_0) CreatedTime() time.Time {
	return p.Created
}

func newCustomPayload() (json.RawMessage, error) {
	p := payloadV1_0{Created: time.Now().UTC()}
	j, err := json.Marshal(&p)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// CustomPayload parses and returns the bundle's custom payload
func CustomPayload(b *bundle.Bundle) (interface{}, error) {
	custom, err := parseCustomPayload(b)
	if err != nil {
		return nil, err
	}

	switch version := custom.Version; version {
	case DockerAppCustomVersion1_0_0:
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
