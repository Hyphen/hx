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

	common "github.com/Hyphen/cli/internal"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/dockerutil"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/httputil"
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

type ConnectionEntity struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type ConnectionOrganizationIntegration struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type AzureContainerRegistryConfiguration struct {
	RegistryId   string `json:"registryId"`
	RegistryName string `json:"registryName"`
	TenantId     string `json:"tenantId"`
	Secrets      struct {
		Auth struct {
			Server            string `json:"server"`
			Username          string `json:"username"`
			EncryptedPassword string `json:"encryptedPassword"`
		} `json:"auth"`
	} `json:"secrets"`
}

type ContainerRegistry struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
	Auth struct {
		Server   string `json:"server"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"auth"`
}

type Connection[T AzureContainerRegistryConfiguration] struct {
	Id                      string                            `json:"id"`
	Type                    string                            `json:"type"`
	Entity                  ConnectionEntity                  `json:"entity"`
	OrganizationIntegration ConnectionOrganizationIntegration `json:"organizationIntegration"`
	Config                  T                                 `json:"config"`
	Status                  string                            `json:"status"`
	Organization            common.OrganizationReference      `json:"organization"`
	Project                 common.ProjectReference           `json:"project"`
}

func (bs *BuildService) CreateBuild(organizationId, appId, environmentId, commitSha, dockerUri string, ports []int) (*Build, error) {

	///api/organizations/{organizationId}/apps/{appId}/builds/
	queryParams := url.Values{}
	if environmentId != "" {
		queryParams.Add("environmentId", environmentId)
	}
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/builds?%s", bs.baseUrl, organizationId, appId, queryParams.Encode())

	build := NewBuild{
		Tags:      []string{},
		CommitSha: commitSha,
		Artifact: Artifact{
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

	var NewBuild Build
	err = json.Unmarshal(body, &NewBuild)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}
	return &NewBuild, nil
}

func (bs *BuildService) FindRegistryConnection(organizationId, projectId string) (*ContainerRegistry, error) {
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

	var response []ContainerRegistry
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

func (bs *BuildService) RunBuild(printer *cprint.CPrinter, environmentId string, verbose bool) (*Build, error) {
	// TODO: this probably shouldn't error if there is no hxkey
	// file it just means they aren't using env or are using the new
	// cert store service.
	// grab the manifest to get app details
	manifest, err := manifest.Restore()
	if err != nil {
		return nil, err
	}

	if manifest.IsMonorepoProject() {
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
	dockerfilePath, err := dockerutil.FindDockerFile()
	if err != nil || dockerfilePath == "" {
		return nil, fmt.Errorf("no docker file found. Dynamic builds are not supported yet")
	}
	printer.PrintVerbose(fmt.Sprintf("found docker file at %s", dockerfilePath))

	// TODO: pull this from a deployment?

	containerRegistry, err := bs.FindRegistryConnection(manifest.OrganizationId, *manifest.ProjectId)
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
	printer.Print(fmt.Sprintf("Building %s", *manifest.AppAlternateId))
	name, _, err := dockerutil.Build(dockerfilePath, *manifest.AppAlternateId, commitSha, verbose)
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
	build, err := bs.CreateBuild(manifest.OrganizationId, *manifest.AppId, environmentId, commitSha, containerUrl, ports)

	if err != nil {
		return nil, fmt.Errorf("failed to create build: %w", err)
	}
	return build, nil

}
