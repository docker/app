package image

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal/slices"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

// Add add service images to the app package
func Add(appname string, services []string, config *composetypes.Config, pull bool, quiet bool) error {
	if err := os.Mkdir(filepath.Join(appname, "images"), 0755); err != nil && !os.IsExist(err) {
		return errors.Wrap(err, "cannot create 'images' folder, this command only works with a multi-file application package")
	}
	for _, s := range config.Services {
		if len(services) != 0 && !slices.ContainsString(services, s.Name) {
			continue
		}
		if !quiet {
			fmt.Printf("Adding image %s for service %s...\n", s.Image, s.Name)
		}
		if pull {
			cmd := exec.Command("docker", "pull", s.Image)
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error pulling %s: %s\n", s.Image, string(output))
				return err
			}
		}
		cmd := exec.Command("docker", "save", "-o", filepath.Join(appname, "images", s.Name), s.Image)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error from docker when saving %s: %s\n", s.Image, string(output))
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
	defer imageDir.Close()
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
			fmt.Println(string(output))
			return err
		}
	}
	return nil
}

func serviceByName(config *composetypes.Config, name string) *composetypes.ServiceConfig {
	for _, s := range config.Services {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

// ChangeAllImages adds given registry to all images used in config
func ChangeAllImages(config *composetypes.Config, registry string) error {
	for i, s := range config.Services {
		ni, err := ChangeImageRepository(s.Image, registry)
		if err != nil {
			return err
		}
		config.Services[i].Image = ni
	}
	return nil
}

// ChangeImageRepository changes the registry for image to given value
func ChangeImageRepository(image, registry string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image reference: %s", err)
	}
	path := reference.Path(named)
	if tagged, ok := named.(reference.Tagged); ok {
		path = path + ":" + tagged.Tag()
	}
	path = registry + "/" + path
	return path, nil
}

func loadImage(image string) (string, error) {
	cmd := exec.Command("docker", "load", "-q", "-i", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return "", err
	}
	// output is one of:
	// Loaded image ID: sha256:cf4d5c8667efcb194163da2da2f3da583602bbd9aa5894e785f51c7f5e91bffb
	// Loaded image: REF
	soutput := string(output)
	var ref string
	if strings.Contains(soutput, "Loaded image:") {
		ref = strings.SplitN(soutput, ":", 2)[1]
	} else {
		outputComponents := strings.Split(string(output), ":")
		if len(outputComponents) != 3 {
			return "", fmt.Errorf("failed to parse 'docker load' output: %s", string(output))
		}
		ref = outputComponents[2]
	}
	ref = strings.Trim(ref, "\n ")
	return ref, nil
}

// Push loads and pushes images found in app to given registry
func Push(appPath string, registry string, services []string, config *composetypes.Config) error {
	imageDir, err := os.Open(filepath.Join(appPath, "images"))
	if err != nil {
		return fmt.Errorf("no images found in app")
	}
	defer imageDir.Close()
	images, err := imageDir.Readdirnames(0)
	if err != nil {
		return err
	}
	for _, i := range images {
		if len(services) != 0 && !slices.ContainsString(services, i) {
			continue
		}
		ref, err := loadImage(filepath.Join(appPath, "images", i))
		if err != nil {
			return err
		}
		service := serviceByName(config, i)
		if service == nil {
			return fmt.Errorf("failed to find service '%s' in config", i)
		}
		path, err := ChangeImageRepository(service.Image, registry)
		if err != nil {
			return err
		}
		// retag
		cmd := exec.Command("docker", "tag", ref, path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to retag '%s' to '%s': %s", ref, path, string(output))
			return err
		}
		// push
		cmd = exec.Command("docker", "push", path)
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to push '%s': %s", path, string(output))
			return err
		}
	}
	return nil
}
