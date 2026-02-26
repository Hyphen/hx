package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/internal/code"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/dockerutil"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/spf13/cobra"
)

type BuildService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() *BuildService {
	baseUrl := apiconf.GetBaseApixUrl()
	return &BuildService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (bs *BuildService) CreateBuild(organizationId, appId, environmentId, commitSha, dockerUri string, ports []int, preview bool) (*models.Build, error) {

	///api/organizations/{organizationId}/apps/{appId}/builds/
	queryParams := url.Values{}
	if environmentId != "" {
		queryParams.Add("environmentId", environmentId)
	}
	if preview {
		queryParams.Add("preview", "true")
	}
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/builds?%s", bs.baseUrl, organizationId, appId, queryParams.Encode())

	build := models.NewBuild{
		Tags:      []string{},
		CommitSha: commitSha,
		Artifact: models.Artifact{
			Type:  "Docker",
			Ports: ports,
			Image: struct {
				URI string `json:"uri"`
			}{
				URI: dockerUri,
			},
		},
	}

	buildJSON, err := json.Marshal(build)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal build data to JSON")
	}

	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewBuffer(buildJSON)))
	if err != nil {
		return nil, err
	}

	resp, err := bs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var NewBuild models.Build
	err = json.Unmarshal(body, &NewBuild)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}
	return &NewBuild, nil
}

func (bs *BuildService) FindRegistryConnection(organizationId, projectId string) (*models.ContainerRegistry, error) {
	///api/organizations/{organizationId}/deployments/containerRegistries
	queryParams := url.Values{}
	queryParams.Add("projectId", projectId)

	hyphenUrl := fmt.Sprintf("%s/api/organizations/%s/deployments/containerRegistries?%s", bs.baseUrl, organizationId, queryParams.Encode())

	req, err := http.NewRequest("GET", hyphenUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := bs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var response []models.ContainerRegistry
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	if len(response) == 0 {
		return nil, fmt.Errorf("no registry connection found")
	}

	// For now we are just going to take the first one,
	// in theory the index will keep this from being more than one
	return &response[0], nil

}

func (bs *BuildService) RunBuild(cmd *cobra.Command, printer *cprint.CPrinter, environmentId string, verbose bool, dockerfilePath string, preview bool) (*models.Build, error) {
	// grab the manifest to get app details
	config, err := config.RestoreConfig()
	if err != nil {
		return nil, err
	}

	if config.IsMonorepoProject() {
		return nil, fmt.Errorf("monorepo projects are not supported yet")
	}

	// Check for docker
	printer.PrintVerbose("Checking for docker CLI")
	isDockerAvailable := dockerutil.IsDockerAvailable()
	if !isDockerAvailable {
		return nil, fmt.Errorf("docker is not installed or not in PATH")
	}

	// Try to find a docker file to run
	printer.PrintVerbose("Looking for Docker File")
	var dockerfilePathOrDir string
	if dockerfilePath != "" {
		// Use the provided dockerfile path (file)
		printer.PrintVerbose(fmt.Sprintf("Using provided dockerfile path: %s", dockerfilePath))
		dockerfilePathOrDir = dockerfilePath
	} else {
		// Search for dockerfile automatically (returns directory)
		dockerfileDir, err := dockerutil.FindDockerFile()
		if err != nil || dockerfileDir == "" {
			coder := code.NewService()
			err = coder.GenerateDocker(printer, cmd)
			if err != nil {
				return nil, fmt.Errorf("failed to generate docker file: %w", err)
			}
			dockerfileDir, _ = dockerutil.FindDockerFile()
		}
		dockerfilePathOrDir = dockerfileDir
	}
	printer.PrintVerbose(fmt.Sprintf("found docker file at %s", dockerfilePathOrDir))

	containerRegistry, err := bs.FindRegistryConnection(config.OrganizationId, *config.ProjectId)
	if err != nil {
		return nil, fmt.Errorf("failed to find registry connection: %w", err)
	}

	// registerUrl := "deploydevelopmentregistry.azurecr.io"
	commitSha, err := gitutil.GetLastCommitHash()
	if err != nil {
		commitSha = "00000000000000000000000000000000"
	}
	commitSha = commitSha[:7]

	// Run build on the docker file
	printer.Print(fmt.Sprintf("Building %s", *config.AppAlternateId))
	name, _, err := dockerutil.Build(dockerfilePathOrDir, *config.AppAlternateId, commitSha, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to build docker image: %w", err)
	}
	printer.PrintVerbose("Docker image built successfully")

	inspectData, err := dockerutil.Inspect(name)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect docker image: %w", err)
	}

	ports := make([]int, 0)
	for port := range inspectData.Config.ExposedPorts {
		// Remove "/tcp" or "/udp" suffix if present
		portStr := port
		if idx := strings.Index(port, "/"); idx != -1 {
			portStr = port[:idx]
		}
		if portInt, err := strconv.Atoi(portStr); err == nil {
			ports = append(ports, portInt)
		}
	}

	// check to see if we need to login into the registry so we don't stomp creds
	needsLogin := !dockerutil.IsLoggedIn(containerRegistry.Auth.Server)

	if needsLogin {
		// make sure we are logged into the registry
		err = dockerutil.Login(containerRegistry.Auth.Server, containerRegistry.Auth.Username, containerRegistry.Auth.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to login to docker registry: %w", err)
		}
		defer dockerutil.Logout(containerRegistry.Auth.Server)
	}

	// push the image to a register
	printer.Print("Uploading artifact")
	containerUrl, err := dockerutil.Push(name, containerRegistry.Url)

	if err != nil {
		return nil, fmt.Errorf("failed to push docker image: %w", err)
	}

	// Tell Hyphen about the build
	build, err := bs.CreateBuild(config.OrganizationId, *config.AppId, environmentId, commitSha, containerUrl, ports, preview)

	if err != nil {
		return nil, fmt.Errorf("failed to create build: %w", err)
	}
	return build, nil

}
