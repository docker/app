package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/spf13/cobra"
	"github.com/xeipuuv/gojsonschema"
)

func main() {
	var index uint

	rootCmd := &cobra.Command{
		Use:   "yamlschema <YAML file> <JSON Schema file>",
		Short: "yamlschema validates a yaml against a json schema",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			yaml, schema := args[0], args[1]
			parsedYaml, err := loadYaml(yaml, index)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			validateYaml(schema, parsedYaml)
		},
	}
	rootCmd.Flags().UintVar(&index, "index", 0, "specify a yaml index if using a multiple document yaml file")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadYaml(file string, index uint) (map[string]interface{}, error) {
	var (
		data []byte
		err  error
	)
	if file == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read YAML from standard input: %s", err)
		}
	} else {
		data, err = ioutil.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read '%s': %s", file, err)
		}
	}
	yamls := bytes.Split(data, []byte("\n---"))
	if index >= uint(len(yamls)) {
		return nil, fmt.Errorf("yaml index '%d' out of bounds '%d'", index, len(yamls))
	}
	return loader.ParseYAML(yamls[int(index)])
}

func validateYaml(schema string, yaml map[string]interface{}) {
	if !strings.HasPrefix(schema, "http") {
		schema = fmt.Sprintf("file://%s", schema)
	}
	schemaLoader := gojsonschema.NewReferenceLoader(schema)
	dataLoader := gojsonschema.NewGoLoader(yaml)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to validate yaml: %s\n", err)
		os.Exit(1)
	}
	if result.Valid() {
		return
	}
	fmt.Println("The document is not valid. See errors :")
	for _, err := range result.Errors() {
		fmt.Printf("- %s\n", err)
	}
	os.Exit(1)
}
