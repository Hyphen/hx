package dockerutil

import (
	"os"
	"os/exec"
	"path/filepath"
)

func IsDockerAvaliable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	return true
}

func FindDockerFile() (string, error) {
	var dockerFileDir string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "Dockerfile" {
			dockerFileDir = filepath.Dir(path) // Get the directory of the Dockerfile
			return filepath.SkipDir            // Stop searching further
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return dockerFileDir, nil
}

func Login(registryUrl, username, password string) error {
	cmd := exec.Command("docker", "login", registryUrl, "-u", username, "-p", password)

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func Build(dockerFilePath, name, tag string, verbose bool) (string, string, error) {
	var nameTag = name
	if tag != "" {
		nameTag = name + ":" + tag
	}
	cmd := exec.Command("docker", "build", "-t", nameTag, dockerFilePath)

	if verbose {
		// TODO: we need to figure out how much we want to hide away
		// perhaps only show this whent there is the verbose flag?
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	if err != nil {
		return "", "", err
	}
	return nameTag, "Build completed successfully", nil
}

func Push(nameTag, registryUrl string) (string, error) {
	// tag the image with the registry URL
	cmdTag := exec.Command("docker", "tag", nameTag, registryUrl+"/"+nameTag)

	_, err := cmdTag.CombinedOutput()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("docker", "push", registryUrl+"/"+nameTag)

	_, err = cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(registryUrl + "/" + nameTag), nil
}
