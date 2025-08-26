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
	cmd := exec.Command("docker", "build", "--platform", "linux/amd64", "-t", nameTag, dockerFilePath)

	if verbose {
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
	var finalImageTag string

	if strings.Contains(registryUrl, ".dkr.ecr.") {
		// AWS: change : to - and use : as separator
		finalImageTag = registryUrl + ":" + strings.ReplaceAll(nameTag, ":", "-")
	} else {
		// GCP/Azure: current behavior
		finalImageTag = registryUrl + "/" + nameTag
	}

	// tag the image with the registry URL
	cmdTag := exec.Command("docker", "tag", nameTag, finalImageTag)

	_, err := cmdTag.CombinedOutput()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("docker", "push", finalImageTag)

	_, err = cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return finalImageTag, nil
}

func Inspect(nameTag string) (DockerInspectResult, error) {
	cmd := exec.Command("docker", "inspect", nameTag)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return DockerInspectResult{}, err
	}

	var inspectData []DockerInspectResult
	if err := json.Unmarshal(output, &inspectData); err != nil {
		return DockerInspectResult{}, err
	}
	if len(inspectData) == 0 {
		return DockerInspectResult{}, nil
	}
	return inspectData[0], nil
}

type DockerInspectResult struct {
	Id            string   `json:"Id"`
	RepoTags      []string `json:"RepoTags"`
	RepoDigests   []string `json:"RepoDigests"`
	Parent        string   `json:"Parent"`
	Comment       string   `json:"Comment"`
	Created       string   `json:"Created"`
	DockerVersion string   `json:"DockerVersion"`
	Author        string   `json:"Author"`
	Config        struct {
		Hostname     string                 `json:"Hostname"`
		Domainname   string                 `json:"Domainname"`
		User         string                 `json:"User"`
		AttachStdin  bool                   `json:"AttachStdin"`
		AttachStdout bool                   `json:"AttachStdout"`
		AttachStderr bool                   `json:"AttachStderr"`
		ExposedPorts map[string]interface{} `json:"ExposedPorts"`
		Tty          bool                   `json:"Tty"`
		OpenStdin    bool                   `json:"OpenStdin"`
		StdinOnce    bool                   `json:"StdinOnce"`
		Env          []string               `json:"Env"`
		Cmd          interface{}            `json:"Cmd"`
		Image        string                 `json:"Image"`
		Volumes      interface{}            `json:"Volumes"`
		WorkingDir   string                 `json:"WorkingDir"`
		Entrypoint   []string               `json:"Entrypoint"`
		OnBuild      interface{}            `json:"OnBuild"`
		Labels       interface{}            `json:"Labels"`
	} `json:"Config"`
	Architecture string `json:"Architecture"`
	Os           string `json:"Os"`
	Size         int64  `json:"Size"`
	GraphDriver  struct {
		Data interface{} `json:"Data"`
		Name string      `json:"Name"`
	} `json:"GraphDriver"`
	RootFS struct {
		Type   string   `json:"Type"`
		Layers []string `json:"Layers"`
	} `json:"RootFS"`
	Metadata struct {
		LastTagTime string `json:"LastTagTime"`
	} `json:"Metadata"`
	Descriptor struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"Descriptor"`
}
