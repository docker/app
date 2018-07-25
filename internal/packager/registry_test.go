package packager

import (
	"testing"

	"gotest.tools/assert"
)

func TestSplitImageName(t *testing.T) {
	input := []string{
		"official.dockerapp",
		"touhou/reimu.dockerapp",
		"tagged.dockerapp:1.2.25",
		"touhou/sakuya.dockerapp:4.23",
		"private.registry.co.uk/docker/anne.boleyn/annulment.dockerapp:15.28",
	}

	output := []imageComponents{
		{Name: "official.dockerapp", Repository: "docker.io/library/official.dockerapp"},
		{Name: "reimu.dockerapp", Repository: "docker.io/touhou/reimu.dockerapp"},
		{Name: "tagged.dockerapp", Repository: "docker.io/library/tagged.dockerapp", Tag: "1.2.25"},
		{Name: "sakuya.dockerapp", Repository: "docker.io/touhou/sakuya.dockerapp", Tag: "4.23"},
		{Name: "annulment.dockerapp", Repository: "private.registry.co.uk/docker/anne.boleyn/annulment.dockerapp", Tag: "15.28"},
	}

	for i, item := range input {
		out, err := splitImageName(item)
		assert.NilError(t, err, item)
		assert.DeepEqual(t, out, &output[i])
	}

	invalids := []string{
		"__.dockerapp",
		"colon:colon:colon.dockerapp:colon",
		"nametag.dockerapp:",
	}

	for _, item := range invalids {
		_, err := splitImageName(item)
		assert.ErrorContains(t, err, "failed to parse image name", item)
	}
}
