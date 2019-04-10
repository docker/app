package remotes

import (
	"encoding/json"

	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type typelessManifestList struct {
	Manifests []typelessDescriptor
	extras    map[string]json.RawMessage
}

func (m *typelessManifestList) MarshalJSON() ([]byte, error) {
	data := map[string]json.RawMessage{}
	for k, v := range m.extras {
		data[k] = v
	}
	if len(m.Manifests) != 0 {
		manifestsJSON, err := json.Marshal(m.Manifests)
		if err != nil {
			return nil, err
		}
		data["manifests"] = json.RawMessage(manifestsJSON)
	}
	return json.Marshal(data)
}

func (m *typelessManifestList) UnmarshalJSON(source []byte) error {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(source, &data); err != nil {
		return err
	}
	if manifestsJSON, ok := data["manifests"]; ok {
		if err := json.Unmarshal(manifestsJSON, &m.Manifests); err != nil {
			return err
		}
		delete(data, "manifests")
	}
	m.extras = data
	return nil
}

type typelessDescriptor struct {
	Platform *ocischemav1.Platform
	extras   map[string]json.RawMessage
}

func (d *typelessDescriptor) MarshalJSON() ([]byte, error) {
	data := map[string]json.RawMessage{}
	for k, v := range d.extras {
		data[k] = v
	}
	if d.Platform != nil {
		platJSON, err := json.Marshal(d.Platform)
		if err != nil {
			return nil, err
		}
		data["platform"] = json.RawMessage(platJSON)
	}
	return json.Marshal(data)
}

func (d *typelessDescriptor) UnmarshalJSON(source []byte) error {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(source, &data); err != nil {
		return err
	}
	if platJSON, ok := data["platform"]; ok {
		var plat ocischemav1.Platform
		if err := json.Unmarshal(platJSON, &plat); err != nil {
			return err
		}
		d.Platform = &plat
		delete(data, "platform")
	}
	d.extras = data
	return nil
}
