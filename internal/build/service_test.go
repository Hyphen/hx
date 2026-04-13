package build

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

		build, err := service.CreateBuild("anOrgId", "anAppId", "theEnvironmentId", "abc1234", "", "", "", "anImage", []int{8080}, "")

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

		build, err := service.CreateBuild("anOrgId", "anAppId", "", "abc1234", "", "", "", "anImage", []int{8080}, "")

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

		build, err := service.CreateBuild(
			"anOrgId", "anAppId", "", "abc1234",
			"https://github.com/owner/repo/commit/abc1234",
			"v1.0.0",
			"https://github.com/owner/repo/releases/tag/v1.0.0",
			"anImage", []int{8080}, "",
		)

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

		build, err := service.CreateBuild("anOrgId", "anAppId", "", "abc1234", "", "", "", "anImage", []int{8080}, "")

		assert.NoError(t, err)
		assert.NotNil(t, build)
		assert.NotContains(t, capturedBody, "commitShaHref")
		assert.NotContains(t, capturedBody, `"tag"`)
		assert.NotContains(t, capturedBody, "tagHref")
		mockHTTPClient.AssertExpectations(t)
	})
}
