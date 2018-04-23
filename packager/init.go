package packager

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"

	"github.com/docker/lunchbox/types"
	"github.com/docker/lunchbox/utils"
	"gopkg.in/yaml.v2"
)

// Init is the entrypoint initialization function.
// It generates a new application package based on the provided parameters.
func Init(name string, composeFiles []string) error {
	if err := utils.ValidateAppName(name); err != nil {
		return err
	}
	dirName := utils.DirNameFromAppName(name)
	if err := os.Mkdir(dirName, 0755); err != nil {
		return err
	}
	if err := writeMetadataFile(name, dirName); err != nil {
		return err
	}

	merger := NewPythonComposeConfigMerger()
	if len(composeFiles) == 0 {
		if _, err := os.Stat("./docker-compose.yml"); os.IsNotExist(err) {
			log.Println("no compose file detected")
			return initFromScratch(name)
		} else if err != nil {
			return err
		}
		return initFromComposeFiles(name, []string{"./docker-compose.yml"}, merger)
	}
	return initFromComposeFiles(name, composeFiles, merger)
}

func initFromScratch(name string) error {
	log.Println("init from scratch")
	fmt.Println(`
Please indicate a list of services that will be used by your application, one per line.
Examples of possible values: java, mysql, redis, ruby, postgres, rabbitmq...`)
	services, err := utils.ReadNewlineSeparatedList(os.Stdin)
	if err != nil {
		return err
	}
	composeData, err := composeFileFromScratch(services)
	if err != nil {
		return err
	}

	dirName := utils.DirNameFromAppName(name)
	if err := utils.CreateFileWithData(path.Join(dirName, "docker-compose.yml"), composeData); err != nil {
		return err
	}
	return utils.CreateFileWithData(path.Join(dirName, "settings.yml"), []byte{'\n'})
}

func initFromComposeFiles(name string, composeFiles []string, merger ComposeConfigMerger) error {
	log.Println("init from compose")

	dirName := utils.DirNameFromAppName(name)
	composeConfig, err := merger.MergeComposeConfig(composeFiles)
	if err != nil {
		return err
	}
	if err = utils.CreateFileWithData(path.Join(dirName, "docker-compose.yml"), composeConfig); err != nil {
		return err
	}
	return utils.CreateFileWithData(path.Join(dirName, "settings.yml"), []byte{'\n'})
}

func composeFileFromScratch(services []string) ([]byte, error) {
	fileStruct := types.NewInitialComposeFile()
	serviceMap := *fileStruct.Services
	for _, svc := range services {
		svcData := utils.MatchService(svc)
		serviceMap[svcData.ServiceName] = types.InitialService{
			Image: svcData.ServiceImage,
		}
	}
	return yaml.Marshal(fileStruct)
}

func writeMetadataFile(name, dirName string) error {
	data, err := yaml.Marshal(newMetadata(name))
	if err != nil {
		return err
	}
	return utils.CreateFileWithData(path.Join(dirName, "metadata.yml"), data)
}

func newMetadata(name string) types.AppMetadata {
	var userName string
	target := types.ApplicationTarget{
		Swarm:      true,
		Kubernetes: true,
	}
	userData, _ := user.Current()
	if userData != nil {
		userName = userData.Username
	}
	info := types.ApplicationInfo{
		Name:   name,
		Labels: []string{"alpha"},
		Author: userName,
	}
	return types.AppMetadata{
		Version:     "0.1.0",
		Targets:     target,
		Application: info,
	}
}

// ComposeConfigMerger is an interface exposing methods to merge
// multiple compose files into one configuration
type ComposeConfigMerger interface {
	MergeComposeConfig(composeFiles []string) ([]byte, error)
}

// PythonComposeConfigMerger implements the ComposeConfigMerger interface and
// executes a `docker-compose` command to merge configs
type PythonComposeConfigMerger struct{}

// NewPythonComposeConfigMerger returns a ComposeConfigMerger implementor
func NewPythonComposeConfigMerger() ComposeConfigMerger {
	return &PythonComposeConfigMerger{}
}

// MergeComposeConfig takes a list of paths and merges the Compose files
// at those paths into a single configuration
func (m *PythonComposeConfigMerger) MergeComposeConfig(composeFiles []string) ([]byte, error) {
	var args []string
	for _, filename := range composeFiles {
		args = append(args, fmt.Sprintf("--file=%v", filename))
	}
	args = append(args, "config")
	cmd := exec.Command("docker-compose", args...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		log.Fatalln(string(err.(*exec.ExitError).Stderr))
	}
	return out, err
}
