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

type CreateBuildOptions struct {
	OrganizationId string
	AppId          string
	EnvironmentId  string
	CommitSha      string
	CommitShaHref  string
	Tag            string
	TagHref        string
	DockerUri      string
	Ports          []int
	Preview        string
	Site           *models.StaticArtifact
}

func (bs *BuildService) CreateBuild(opts CreateBuildOptions) (*models.Build, error) {

	///api/organizations/{organizationId}/apps/{appId}/builds/
	queryParams := url.Values{}
	if opts.EnvironmentId != "" {
		queryParams.Add("environmentId", opts.EnvironmentId)
	}
	if opts.Preview != "" {
		queryParams.Add("previewName", opts.Preview)
	}
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/builds?%s", bs.baseUrl, opts.OrganizationId, opts.AppId, queryParams.Encode())

	build := models.NewBuild{
		Tags:          []string{},
		CommitSha:     opts.CommitSha,
		CommitShaHref: opts.CommitShaHref,
		Tag:           opts.Tag,
		TagHref:       opts.TagHref,
		Artifact: models.Artifact{
			Type:  "Docker",
			Ports: opts.Ports,
			Image: &struct {
				URI string `json:"uri"`
			}{
				URI: opts.DockerUri,
			},
		},
	}
	if opts.Site != nil {
		build.Artifact = models.Artifact{Type: "Static", Target: "hyphenCloud", Site: opts.Site}
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

func (bs *BuildService) FindRegistryConnections(organizationId, projectId string) ([]models.ContainerRegistry, error) {
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
		return nil, fmt.Errorf("no registry connections found")
	}

	return response, nil
}

type Options struct {
	Type           string
	Directory      string
	EnvironmentID  string
	Preview        string
	PreviewHash    string
	DockerfilePath string
	Verbose        bool
}

func (bs *BuildService) RunBuild(cmd *cobra.Command, printer *cprint.CPrinter, opts Options) (*models.Build, error) {
	if err := ValidateInput(opts.Type, opts.Directory, opts.DockerfilePath); err != nil {
		return nil, err
	}
	// grab the manifest to get app details
	config, err := config.RestoreConfig()
	if err != nil {
		return nil, err
	}

	if config.IsMonorepoProject() {
		return nil, fmt.Errorf("monorepo projects are not supported yet")
	}
	if config.ProjectId == nil || config.AppId == nil || config.AppAlternateId == nil {
		return nil, fmt.Errorf("project and app must be set in .hx configuration")
	}
	metadata := sourceMetadata()
	metadata.OrganizationId = config.OrganizationId
	metadata.AppId = *config.AppId
	metadata.EnvironmentId = opts.EnvironmentID
	metadata.Preview = opts.Preview
	if opts.Type == "static" {
		site, err := bs.uploadStaticSite(cmd.Context(), printer, config.OrganizationId, *config.ProjectId, *config.AppId, opts)
		if err != nil {
			return nil, err
		}
		metadata.Site = site
	} else {
		uri, ports, err := bs.buildDocker(cmd, printer, config, opts, metadata.CommitSha)
		if err != nil {
			return nil, err
		}
		metadata.DockerUri, metadata.Ports = uri, ports
	}
	result, err := bs.CreateBuild(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to register build: %w", err)
	}
	return result, nil
}

func (bs *BuildService) buildDocker(cmd *cobra.Command, printer *cprint.CPrinter, config config.Config, opts Options, commitSha string) (string, []int, error) {

	// Check for docker
	printer.PrintVerbose("Checking for docker CLI")
	isDockerAvailable := dockerutil.IsDockerAvailable()
	if !isDockerAvailable {
		return "", nil, fmt.Errorf("docker is not installed or not in PATH")
	}

	// Try to find a docker file to run
	printer.PrintVerbose("Looking for Docker File")
	var dockerfilePathOrDir string
	if opts.DockerfilePath != "" {
		// Use the provided dockerfile path (file)
		printer.PrintVerbose(fmt.Sprintf("Using provided dockerfile path: %s", opts.DockerfilePath))
		dockerfilePathOrDir = opts.DockerfilePath
	} else {
		// Search for dockerfile automatically (returns directory)
		dockerfileDir, err := dockerutil.FindDockerFile()
		if err != nil || dockerfileDir == "" {
			coder := code.NewService()
			err = coder.GenerateDocker(printer, cmd)
			if err != nil {
				return "", nil, fmt.Errorf("failed to generate docker file: %w", err)
			}
			dockerfileDir, _ = dockerutil.FindDockerFile()
		}
		dockerfilePathOrDir = dockerfileDir
	}
	printer.PrintVerbose(fmt.Sprintf("found docker file at %s", dockerfilePathOrDir))

	containerRegistries, err := bs.FindRegistryConnections(config.OrganizationId, *config.ProjectId)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find registry connections: %w", err)
	}

	// Run build on the docker file
	printer.Print(fmt.Sprintf("Building %s", *config.AppAlternateId))
	name, _, err := dockerutil.Build(dockerfilePathOrDir, *config.AppAlternateId, commitSha, opts.Verbose)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build docker image: %w", err)
	}
	printer.PrintVerbose("Docker image built successfully")

	inspectData, err := dockerutil.Inspect(name)
	if err != nil {
		return "", nil, fmt.Errorf("failed to inspect docker image: %w", err)
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

	var containerUrl string
	for _, containerRegistry := range containerRegistries {
		err := func(containerRegistry models.ContainerRegistry) error {
			registryLabel := containerRegistry.Name
			if registryLabel == "" {
				registryLabel = containerRegistry.Url
			}

			// check to see if we need to login into the registry so we don't stomp creds
			needsLogin := !dockerutil.IsLoggedIn(containerRegistry.Auth.Server)
			if needsLogin {
				err := dockerutil.Login(containerRegistry.Auth.Server, containerRegistry.Auth.Username, containerRegistry.Auth.Password)
				if err != nil {
					return fmt.Errorf("failed to login to docker registry %s: %w", registryLabel, err)
				}
				defer func() {
					_ = dockerutil.Logout(containerRegistry.Auth.Server)
				}()
			}

			printer.Print(fmt.Sprintf("Uploading artifact to %s", registryLabel))
			pushedUrl, err := dockerutil.Push(name, containerRegistry.Url)
			if err != nil {
				return fmt.Errorf("failed to push docker image to %s: %w", registryLabel, err)
			}

			// Register the first registry's URI with the build API
			if containerUrl == "" {
				containerUrl = pushedUrl
			}
			return nil
		}(containerRegistry)
		if err != nil {
			return "", nil, err
		}
	}
	return containerUrl, ports, nil
}

func sourceMetadata() CreateBuildOptions {
	fullCommitSha, err := gitutil.GetLastCommitHash()
	if err != nil {
		fullCommitSha = "00000000000000000000000000000000"
	}
	opts := CreateBuildOptions{CommitSha: fullCommitSha[:7]}
	opts.Tag, _ = gitutil.GetCurrentTag()
	remoteURL, _ := gitutil.GetRemoteUrl()
	baseURL, _ := gitutil.ParseRepoBaseUrl(remoteURL)
	if baseURL != "" {
		var commitPath, tagPath string
		switch gitutil.DetectProvider(remoteURL) {
		case "github":
			commitPath, tagPath = "/commit/", "/releases/tag/"
		case "gitlab":
			commitPath, tagPath = "/-/commit/", "/-/tags/"
		case "azuredevops":
			commitPath, tagPath = "/commit/", "?version=GT"
		case "bitbucket":
			commitPath, tagPath = "/commits/", "/src/"
		}
		if commitPath != "" {
			opts.CommitShaHref = baseURL + commitPath + fullCommitSha
		}
		if tagPath != "" && opts.Tag != "" {
			opts.TagHref = baseURL + tagPath + opts.Tag
		}
	}
	return opts
}
