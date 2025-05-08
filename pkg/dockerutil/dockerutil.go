package dockerutil

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func IsDockerAvailable() bool {
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
		if strings.ToLower(info.Name()) == "dockerfile" {
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

func Logout(registryUrl string) error {
	cmd := exec.Command("docker", "logout", registryUrl)

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func IsLoggedIn(registryUrl string) bool {
	// Get the path to the Docker config file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	dockerConfigPath := filepath.Join(homeDir, ".docker", "config.json")

	// Open the Docker config file
	file, err := os.Open(dockerConfigPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Parse the JSON content
	var config map[string]interface{}
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return false
	}

	// Check for the "auths" property and the registry URL
	auths, ok := config["auths"].(map[string]interface{})
	if !ok {
		return false
	}

	if _, exists := auths[registryUrl]; exists {
		return true
	}

	return false
}

func Build(dockerFilePath, name, tag string, verbose bool) (string, string, error) {
	var nameTag = name
	if tag != "" {
		nameTag = name + ":" + tag
	}
	// TODO: consider allowing platform to be passed in. However, the most common
	// supported platform in the clouds is linux/amd64
	cmd := exec.Command("docker", "build", "--platform linux/amd64", "-t", nameTag, dockerFilePath)

	if verbose {
		// TODO: we need to figure out how much we want to hide away
		// perhaps only show this when there is the verbose flag?
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
