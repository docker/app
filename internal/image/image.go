package image

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/slices"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

// Add add service images to the app package
func Add(appname string, services []string, config *composetypes.Config) error {
	if err := os.Mkdir(filepath.Join(appname, "images"), 0755); err != nil {
		return errors.Wrap(err, "cannot create 'images' folder")
	}
	for _, s := range config.Services {
		if len(services) != 0 && !slices.ContainsString(services, s.Name) {
			continue
		}
		cmd := exec.Command("docker", "save", "-o", filepath.Join(appname, "images", s.Name), s.Image)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(output)
			return err
		}
	}
	return nil
}

// Load loads app packed images to the docker daemon
func Load(appname string, services []string) error {
	imageDir, err := os.Open(filepath.Join(appname, "images"))
	if err != nil {
		return fmt.Errorf("no images found in app")
	}
	images, err := imageDir.Readdirnames(0)
	if err != nil {
		return err
	}
	for _, i := range images {
		if len(services) != 0 && !slices.ContainsString(services, i) {
			continue
		}
		cmd := exec.Command("docker", "load", "-i", filepath.Join(appname, "images", i))
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(output)
			return err
		}
	}
	return nil
}

// List images with the label specific to applications.
func List(appname string, quiet bool) error {
	args := []string{
		"image", "ls", "--filter", fmt.Sprintf("label=%s", internal.ImageLabel),
	}
	if quiet {
		args = append(args, "-q")
	}
	args = append(args, []string{"--", appname}...)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
