package packager

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/types"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

func TestToCNAB(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	actual, err := ToCNAB(app, "test-image")
	assert.NilError(t, err)
	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	assert.NilError(t, err)
	golden.Assert(t, string(actualJSON), "bundle-json.golden")
}

func TestCnabAutomaticParameters(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/auto-parameters.dockerapp")
	assert.NilError(t, err)
	actual, err := ToCNAB(app, "test-image")
	assert.NilError(t, err)
	checkOverrideParameter(t, actual, "services.nothing-specified.deploy.replicas")
	checkOverrideParameter(t, actual, "services.nothing-specified.deploy.resources.limits.cpus")
	checkOverrideParameter(t, actual, "services.nothing-specified.deploy.resources.limits.memory")
	checkOverrideParameter(t, actual, "services.nothing-specified.deploy.resources.reservations.cpus")
	checkOverrideParameter(t, actual, "services.nothing-specified.deploy.resources.reservations.memory")
	checkNoParameter(t, actual, "services.replicas-fixed.deploy.replicas")
	checkOverrideParameter(t, actual, "services.replicas-fixed.deploy.resources.limits.cpus")
	checkOverrideParameter(t, actual, "services.replicas-fixed.deploy.resources.limits.memory")
	checkOverrideParameter(t, actual, "services.replicas-fixed.deploy.resources.reservations.cpus")
	checkOverrideParameter(t, actual, "services.replicas-fixed.deploy.resources.reservations.memory")
	checkCustomParameter(t, actual, "services.parameter-names-used.deploy.replicas")
	checkCustomParameter(t, actual, "services.parameter-names-used.deploy.resources.limits.cpus")
	checkCustomParameter(t, actual, "services.parameter-names-used.deploy.resources.limits.memory")
	checkCustomParameter(t, actual, "services.parameter-names-used.deploy.resources.reservations.cpus")
	checkCustomParameter(t, actual, "services.parameter-names-used.deploy.resources.reservations.memory")
}

func checkOverrideParameter(t *testing.T, b *bundle.Bundle, parameterName string) {
	t.Helper()
	parameterDest := "/cnab/app/overrides/" + strings.ReplaceAll(parameterName, ".", "/")
	param, ok := b.Parameters[parameterName]
	if !ok {
		t.Fatalf("parameter %q is not present", parameterName)
	}
	assert.Check(t, param.Destination != nil)
	assert.Equal(t, param.Destination.Path, parameterDest)
}

func checkNoParameter(t *testing.T, b *bundle.Bundle, parameterName string) {
	t.Helper()
	_, ok := b.Parameters[parameterName]
	if ok {
		t.Fatalf("parameter %q is present", parameterName)
	}
}

func checkCustomParameter(t *testing.T, b *bundle.Bundle, parameterName string) {
	t.Helper()
	param, ok := b.Parameters[parameterName]
	if !ok {
		t.Fatalf("parameter %q is not present", parameterName)
	}
	assert.Check(t, param.Destination != nil)
	assert.Equal(t, param.Destination.Path, "")
}
