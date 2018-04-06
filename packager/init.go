package packager

import (
    "fmt"
    "github.com/docker/lunchbox/types"
    "github.com/docker/lunchbox/utils"
    "gopkg.in/yaml.v2"
    "log"
    "os"
    "os/exec"
    "os/user"
    "path"
)

// Init is the entrypoint initialization function.
// It generates a new application package based on the provided parameters.
func Init(name string, composeFiles []string) error {
    if err := os.Mkdir(appDirName(name), 0755); err != nil {
        return err
    }

    if len(composeFiles) == 0 {
        if _, err := os.Stat("./docker-compose.yml"); os.IsNotExist(err) {
            return initFromScratch(name)
        }
        return initFromComposeFiles(name, []string{"./docker-compose.yml"})
    }
    return initFromComposeFiles(name, composeFiles)
}

func initFromScratch(name string) error {
    log.Println("init from scratch")

    dirName := appDirName(name)
    if err := writeMetadataFile(name, dirName); err != nil {
        return err
    }
    if err := utils.CreateFileWithData(path.Join(dirName, "services.yml"), []byte{'\n'}); err != nil {
        return err
    }
    return utils.CreateFileWithData(path.Join(dirName, "settings.yml"), []byte{'\n'})
}

func initFromComposeFiles(name string, composeFiles []string) error {
    log.Println("init from compose")
    dirName := appDirName(name)
    if err := writeMetadataFile(name, dirName); err != nil {
        return err
    }
    composeConfig, err := mergeComposeConfig(composeFiles)
    if err != nil {
        return err
    }
    if err = utils.CreateFileWithData(path.Join(dirName, "services.yml"), composeConfig); err != nil {
        return err
    }
    return utils.CreateFileWithData(path.Join(dirName, "settings.yml"), []byte{'\n'})
}

func mergeComposeConfig(composeFiles []string) ([]byte, error) {
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

func appDirName(name string) string {
    return fmt.Sprintf("%s.docker-app", name)
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
        Version:     "0.0.1",
        Targets:     target,
        Application: info,
    }
}
