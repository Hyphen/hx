package build

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFindRegistryConnections(t *testing.T) {
	t.Run("returns_all_registries_when_multiple_are_present", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &BuildService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		var capturedURL string
		mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
			req := args.Get(0).(*http.Request)
			capturedURL = req.URL.String()
		}).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[
				{"id":"reg1","name":"azure-registry","url":"myregistry.azurecr.io","auth":{"server":"myregistry.azurecr.io","username":"user1","password":"pass1"}},
				{"id":"reg2","name":"aws-registry","url":"123.dkr.ecr.us-east-1.amazonaws.com/repo","auth":{"server":"123.dkr.ecr.us-east-1.amazonaws.com","username":"AWS","password":"token"}}
			]`)),
		}, nil)

		registries, err := service.FindRegistryConnections("anOrgId", "aProjectId")

		assert.NoError(t, err)
		require.Len(t, registries, 2)
		assert.Equal(t, "azure-registry", registries[0].Name)
		assert.Equal(t, "myregistry.azurecr.io", registries[0].Url)
		assert.Equal(t, "aws-registry", registries[1].Name)
		assert.Equal(t, "123.dkr.ecr.us-east-1.amazonaws.com/repo", registries[1].Url)
		assert.Contains(t, capturedURL, "/api/organizations/anOrgId/deployments/containerRegistries")
		assert.Contains(t, capturedURL, "projectId=aProjectId")
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("returns_error_when_no_registries_are_found", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &BuildService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`[]`)),
		}, nil)

		registries, err := service.FindRegistryConnections("anOrgId", "aProjectId")

		assert.Nil(t, registries)
		assert.EqualError(t, err, "no registry connections found")
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("returns_single_registry_when_only_one_is_present", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &BuildService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[
				{"id":"reg1","name":"azure-registry","url":"myregistry.azurecr.io","auth":{"server":"myregistry.azurecr.io","username":"user1","password":"pass1"}}
			]`)),
		}, nil)

		registries, err := service.FindRegistryConnections("anOrgId", "aProjectId")

		assert.NoError(t, err)
		require.Len(t, registries, 1)
		assert.Equal(t, "azure-registry", registries[0].Name)
		mockHTTPClient.AssertExpectations(t)
	})
}

func TestCreateBuild(t *testing.T) {
	t.Run("includes_environmentId_query_param_when_environmentId_is_provided", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &BuildService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		var capturedURL string
		mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
			req := args.Get(0).(*http.Request)
			capturedURL = req.URL.String()
		}).Return(&http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"theBuildId","organization":{"id":"anOrgId","name":"anOrg"},"project":{"id":"aProjectId","name":"aProject","alternateId":"aProject"},"projectEnvironment":{"id":"theEnvId","name":"anEnv"},"app":{"id":"anAppId","name":"anApp","alternateId":"anApp"},"tags":[],"commitSha":"abc1234","artifact":{"type":"Docker","ports":[8080],"image":{"uri":"anImage"}}}`)),
		}, nil)

		build, err := service.CreateBuild(CreateBuildOptions{
			OrganizationId: "anOrgId",
			AppId:          "anAppId",
			EnvironmentId:  "theEnvironmentId",
			CommitSha:      "abc1234",
			DockerUri:      "anImage",
			Ports:          []int{8080},
		})

		assert.NoError(t, err)
		assert.NotNil(t, build)
		assert.Contains(t, capturedURL, "environmentId=theEnvironmentId")
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("does_not_include_environmentId_query_param_when_environmentId_is_empty", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &BuildService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		var capturedURL string
		mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
			req := args.Get(0).(*http.Request)
			capturedURL = req.URL.String()
		}).Return(&http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"aBuildId","organization":{"id":"anOrgId","name":"anOrg"},"project":{"id":"aProjectId","name":"aProject","alternateId":"aProject"},"projectEnvironment":{"id":"","name":""},"app":{"id":"anAppId","name":"anApp","alternateId":"anApp"},"tags":[],"commitSha":"abc1234","artifact":{"type":"Docker","ports":[8080],"image":{"uri":"anImage"}}}`)),
		}, nil)

		build, err := service.CreateBuild(CreateBuildOptions{
			OrganizationId: "anOrgId",
			AppId:          "anAppId",
			CommitSha:      "abc1234",
			DockerUri:      "anImage",
			Ports:          []int{8080},
		})

		assert.NoError(t, err)
		assert.NotNil(t, build)
		assert.NotContains(t, capturedURL, "environmentId")
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("sends_commitShaHref_tag_and_tagHref_in_request_body_when_provided", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &BuildService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		var capturedBody string
		mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
			req := args.Get(0).(*http.Request)
			bodyBytes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			capturedBody = string(bodyBytes)
		}).Return(&http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"aBuildId","organization":{"id":"anOrgId","name":"anOrg"},"project":{"id":"aProjectId","name":"aProject","alternateId":"aProject"},"projectEnvironment":{"id":"","name":""},"app":{"id":"anAppId","name":"anApp","alternateId":"anApp"},"tags":[],"commitSha":"abc1234","commitShaHref":"https://github.com/owner/repo/commit/abc1234","tag":"v1.0.0","tagHref":"https://github.com/owner/repo/releases/tag/v1.0.0","artifact":{"type":"Docker","ports":[8080],"image":{"uri":"anImage"}}}`)),
		}, nil)

		build, err := service.CreateBuild(CreateBuildOptions{
			OrganizationId: "anOrgId",
			AppId:          "anAppId",
			CommitSha:      "abc1234",
			CommitShaHref:  "https://github.com/owner/repo/commit/abc1234",
			Tag:            "v1.0.0",
			TagHref:        "https://github.com/owner/repo/releases/tag/v1.0.0",
			DockerUri:      "anImage",
			Ports:          []int{8080},
		})

		assert.NoError(t, err)
		assert.NotNil(t, build)
		assert.Contains(t, capturedBody, `"commitShaHref":"https://github.com/owner/repo/commit/abc1234"`)
		assert.Contains(t, capturedBody, `"tag":"v1.0.0"`)
		assert.Contains(t, capturedBody, `"tagHref":"https://github.com/owner/repo/releases/tag/v1.0.0"`)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("omits_commitShaHref_tag_and_tagHref_from_request_body_when_empty", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &BuildService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		var capturedBody string
		mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
			req := args.Get(0).(*http.Request)
			bodyBytes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			capturedBody = string(bodyBytes)
		}).Return(&http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"aBuildId","organization":{"id":"anOrgId","name":"anOrg"},"project":{"id":"aProjectId","name":"aProject","alternateId":"aProject"},"projectEnvironment":{"id":"","name":""},"app":{"id":"anAppId","name":"anApp","alternateId":"anApp"},"tags":[],"commitSha":"abc1234","artifact":{"type":"Docker","ports":[8080],"image":{"uri":"anImage"}}}`)),
		}, nil)

		build, err := service.CreateBuild(CreateBuildOptions{
			OrganizationId: "anOrgId",
			AppId:          "anAppId",
			CommitSha:      "abc1234",
			DockerUri:      "anImage",
			Ports:          []int{8080},
		})

		assert.NoError(t, err)
		assert.NotNil(t, build)
		assert.NotContains(t, capturedBody, "commitShaHref")
		assert.NotContains(t, capturedBody, `"tag"`)
		assert.NotContains(t, capturedBody, "tagHref")
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("passes_through_commitShaHref_and_tagHref_values_unchanged_in_request_body_for_each_provider", func(t *testing.T) {
		tests := []struct {
			provider      string
			commitShaHref string
			tagHref       string
		}{
			{
				provider:      "github",
				commitShaHref: "https://github.com/owner/repo/commit/abc1234def5678901234567890123456789012345",
				tagHref:       "https://github.com/owner/repo/releases/tag/v1.0.0",
			},
			{
				provider:      "gitlab",
				commitShaHref: "https://gitlab.com/owner/repo/-/commit/abc1234def5678901234567890123456789012345",
				tagHref:       "https://gitlab.com/owner/repo/-/tags/v1.0.0",
			},
			{
				provider:      "azuredevops",
				commitShaHref: "https://dev.azure.com/org/project/_git/repo/commit/abc1234def5678901234567890123456789012345",
				tagHref:       "https://dev.azure.com/org/project/_git/repo?version=GTv1.0.0",
			},
			{
				provider:      "bitbucket",
				commitShaHref: "https://bitbucket.org/owner/repo/commits/abc1234def5678901234567890123456789012345",
				tagHref:       "https://bitbucket.org/owner/repo/src/v1.0.0",
			},
		}

		for _, tt := range tests {
			t.Run(tt.provider, func(t *testing.T) {
				mockHTTPClient := new(httputil.MockHTTPClient)
				service := &BuildService{
					baseUrl:    "https://api.example.com",
					httpClient: mockHTTPClient,
				}

				var capturedBody string
				mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)
					bodyBytes, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					capturedBody = string(bodyBytes)
				}).Return(&http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"aBuildId","organization":{"id":"anOrgId","name":"anOrg"},"project":{"id":"aProjectId","name":"aProject","alternateId":"aProject"},"projectEnvironment":{"id":"","name":""},"app":{"id":"anAppId","name":"anApp","alternateId":"anApp"},"tags":[],"commitSha":"abc1234","artifact":{"type":"Docker","ports":[8080],"image":{"uri":"anImage"}}}`)),
				}, nil)

				build, err := service.CreateBuild(CreateBuildOptions{
					OrganizationId: "anOrgId",
					AppId:          "anAppId",
					CommitSha:      "abc1234",
					CommitShaHref:  tt.commitShaHref,
					Tag:            "v1.0.0",
					TagHref:        tt.tagHref,
					DockerUri:      "anImage",
					Ports:          []int{8080},
				})

				assert.NoError(t, err)
				assert.NotNil(t, build)
				assert.Contains(t, capturedBody, fmt.Sprintf(`"commitShaHref":"%s"`, tt.commitShaHref))
				assert.Contains(t, capturedBody, `"tag":"v1.0.0"`)
				assert.Contains(t, capturedBody, fmt.Sprintf(`"tagHref":"%s"`, tt.tagHref))
				mockHTTPClient.AssertExpectations(t)
			})
		}
	})
}

func TestSelectBuildRegistry(t *testing.T) {
	gcp := models.ContainerRegistry{Id: "gcp", Url: "us-docker.pkg.dev/project/images"}
	azure := models.ContainerRegistry{Id: "azure", Url: "project.azurecr.io"}
	cases := []struct {
		name                         string
		registries                   []models.ContainerRegistry
		selector, wantURL, wantError string
	}{
		{"single registry", []models.ContainerRegistry{gcp}, "", gcp.Url, ""},
		{"explicit URL", []models.ContainerRegistry{gcp, azure}, azure.Url, azure.Url, ""},
		{"explicit ID", []models.ContainerRegistry{azure, gcp}, gcp.Id, gcp.Url, ""},
		{"selection is independent of order", []models.ContainerRegistry{gcp, azure}, gcp.Id, gcp.Url, ""},
		{"multiple requires selection", []models.ContainerRegistry{gcp, azure}, "", "", "choose the build source with --registry"},
		{"reversed order requires selection", []models.ContainerRegistry{azure, gcp}, "", "", "choose the build source with --registry"},
		{"unknown selection", []models.ContainerRegistry{gcp}, "other", "", "not a ready registry for this project"},
		{"ambiguous selector", []models.ContainerRegistry{gcp, gcp}, gcp.Id, "", "choose the build source with --registry"},
		{"no registries", nil, "", "", "no registry connections found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := append([]models.ContainerRegistry(nil), tc.registries...)
			registry, err := selectBuildRegistry(tc.registries, tc.selector)
			assert.Equal(t, original, tc.registries, "source selection must not filter or reorder upload destinations")
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				assert.Nil(t, registry)
			} else {
				require.NoError(t, err)
				require.NotNil(t, registry)
				assert.Equal(t, tc.wantURL, registry.Url)
			}
		})
	}
}
