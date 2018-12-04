package app

import (
	"fmt"
	"os"

	"github.com/docker/app/loader"
	"github.com/docker/app/render"
	yaml "gopkg.in/yaml.v2"
)

func Example() {
	// Load the file (single-file format, there is multiple format)
	f, err := os.Open("./examples/hello-world/hello-world.dockerapp")
	if err != nil {
		panic("cannot read application")
	}
	defer f.Close()
	app, err := loader.LoadFromSingleFile("myApp", f)
	if err != nil {
		panic("cannot load application")
	}
	// Render the app to a composefile format, using some user provided parameters
	c, err := render.Render(app, map[string]string{
		"text": "hello examples!",
	})
	if err != nil {
		panic("cannot render application")
	}
	// Marshal it to yaml (to display it)
	s, err := yaml.Marshal(c)
	if err != nil {
		panic("cannot marshall the composefile in yaml")
	}
	fmt.Print(string(s))
	// Output: version: "3.6"
	// services:
	//   hello:
	//     command:
	//     - -text
	//     - hello examples!
	//     image: hashicorp/http-echo
	//     ports:
	//     - mode: ingress
	//       target: 5678
	//       published: 8080
	//       protocol: tcp
}
