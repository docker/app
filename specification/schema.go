package specification

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

//go:generate esc -o bindata.go -pkg specification -ignore .*\.go -private -modtime=1518458244 schemas

// Validate uses the jsonschema to validate the configuration
func Validate(config map[string]interface{}, version string) error {
	schemaData, err := _escFSByte(false, fmt.Sprintf("/schemas/metadata_schema_%s.json", version))
	if err != nil {
		return errors.Errorf("unsupported metadata version: %s", version)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaData))
	dataLoader := gojsonschema.NewGoLoader(config)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		errs := make([]string, len(result.Errors()))
		for i, err := range result.Errors() {
			errs[i] = fmt.Sprintf("- %s", err)
		}
		sort.Strings(errs)
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}
