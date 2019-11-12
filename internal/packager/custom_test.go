package packager

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"gotest.tools/assert"
)

func TestNewCustomPayload(t *testing.T) {
	start := time.Now().UTC()
	payloadJSON, err := newCustomPayload()
	assert.NilError(t, err)

	var payload payloadV1_0
	err = json.Unmarshal(payloadJSON, &payload)
	assert.NilError(t, err)

	end := time.Now().UTC()
	assert.Assert(t, start.Before(payload.CreatedTime()) || start.Equal(payload.CreatedTime()))
	assert.Assert(t, end.After(payload.CreatedTime()) || end.Equal(payload.CreatedTime()))
}

func TestCustomPayloadNil(t *testing.T) {
	testCases := []struct {
		testName string
		version  string
		payload  interface{}
	}{
		{
			testName: "NoVersion",
			version:  "",
			payload:  payloadV1_0{},
		},
		{
			testName: "UnknownVersion",
			version:  "unknown-version",
			payload:  payloadV1_0{},
		},
		{
			testName: "NoPayload",
			version:  DockerAppCustomVersionCurrent,
			payload:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			b := createBundle(t, "", payloadV1_0{})
			payload, err := CustomPayload(&b)
			assert.NilError(t, err)
			assert.Assert(t, payload == nil)
		})
	}
}

func TestCustomPayloadV1_0_0(t *testing.T) {
	now := time.Now().UTC()
	b := createBundle(t, DockerAppCustomVersion1_0_0, payloadV1_0{now})
	payload, err := CustomPayload(&b)
	assert.NilError(t, err)
	v1, ok := payload.(payloadV1_0)
	assert.Assert(t, ok)
	assert.Assert(t, now.Equal(v1.CreatedTime()))
}

func createBundle(t *testing.T, version string, payload interface{}) bundle.Bundle {
	j, err := json.Marshal(payload)
	assert.NilError(t, err)
	return bundle.Bundle{
		Custom: map[string]interface{}{
			internal.CustomDockerAppName: DockerAppCustom{
				Version: version,
				Payload: j,
			},
		},
	}
}
