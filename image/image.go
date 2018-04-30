package image

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/lunchbox/packager"
	"github.com/docker/lunchbox/renderer"
	"github.com/docker/lunchbox/utils"
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
	oappname := appname
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
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
	// check if source was a tarball
	s, err := os.Stat(oappname)
	if err != nil {
		// try appending our extension
		oappname = utils.DirNameFromAppName(oappname)
		s, err = os.Stat(oappname)
	}
	if err != nil {
		return err // this shouldn't happen
	}
	if !s.IsDir() {
		// source was a tarball, rebuild it
		return packager.Pack(appname, oappname)
	}
	return nil
}

// Load loads app packed images to the docker daemon
func Load(appname string, services []string) error {
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
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
