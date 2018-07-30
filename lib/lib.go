package lib

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/render"
	yaml "gopkg.in/yaml.v2"
)

// Render renders the application into a Compose file.
func Render(appname string, settingsFiles []string, settings map[string]string) ([]byte, error) {
	app, err := packager.Extract(appname)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()
	rendered, err := render.Render(app.AppName, nil, settingsFiles, settings)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(rendered)
}
