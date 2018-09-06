package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/app/internal/yaml"
	"github.com/docker/app/pkg/yatee"
)

func main() {
	if len(os.Args) == 1 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Printf("usage: %s TEMPLATEFILE SETTINGSFILES...\n", os.Args[0])
		os.Exit(1)
	}
	input, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	settings, err := yatee.LoadSettings(os.Args[2:])
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	output, err := yatee.Process(string(input), settings)
	if err != nil {
		fmt.Printf("processing error: %v\n", err)
		os.Exit(1)
	}
	raw, err := yaml.Marshal(output)
	if err != nil {
		fmt.Printf("marshalling error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(raw))
}
