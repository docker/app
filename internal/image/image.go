package image

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/app/internal/renderer"
)

func contains(list []string, needle string) bool {
	for _, e := range list {
		if e == needle {
			return true
		}
	}
	return false
}

// Add add service images to the app package
func Add(appname string, services []string, composeFiles []string, settingsFile []string, env map[string]string) error {
	config, err := renderer.Render(appname, composeFiles, settingsFile, env)
	if err != nil {
		return err
	}
	os.Mkdir(filepath.Join(appname, "images"), 0755)
	for _, s := range config.Services {
		if len(services) != 0 && !contains(services, s.Name) {
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
		if len(services) != 0 && !contains(services, i) {
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
func List(appname string) error {
	cmd := exec.Command(
		"docker", "image", "ls",
		"--filter", fmt.Sprintf("label=%s", internal.ImageLabel),
		"--", appname)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
