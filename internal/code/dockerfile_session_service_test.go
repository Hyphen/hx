package code

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDockerfileSessionService_StartSession(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &DockerfileSessionService{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	var capturedURL string
	var capturedMethod string
	var capturedBody string

	mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
		req := args.Get(0).(*http.Request)
		capturedURL = req.URL.String()
		capturedMethod = req.Method
		body, _ := io.ReadAll(req.Body)
		capturedBody = string(body)
	}).Return(&http.Response{
		StatusCode: http.StatusCreated,
		Body: io.NopCloser(strings.NewReader(`{
			"session": {"id": "session-1", "status": "requires_tool_results"},
			"output": {
				"session_id": "session-1",
				"status": "requires_tool_results",
				"message": {
					"role": "assistant",
					"content": null,
					"tool_calls": [{
						"id": "call-1",
						"type": "function",
						"function": {"name": "list_files", "arguments": "{\"path\":\".\"}"}
					}]
				}
			}
		}`)),
	}, nil)

	response, err := service.StartSession("org-1", "app-1")

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "https://api.example.com/api/organizations/org-1/apps/app-1/dockerfile", capturedURL)
	assert.JSONEq(t, `{}`, capturedBody)
	assert.Equal(t, "session-1", response.Session.ID)
	assert.Equal(t, DockerfileSessionStatusRequiresToolResults, response.Output.Status)
	assert.Equal(t, "list_files", response.Output.Message.ToolCalls[0].Function.Name)
	mockHTTPClient.AssertExpectations(t)
}

func TestDockerfileSessionService_ContinueSession(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &DockerfileSessionService{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	var capturedBody string

	mockHTTPClient.On("Do", mock.Anything).Run(func(args mock.Arguments) {
		req := args.Get(0).(*http.Request)
		body, _ := io.ReadAll(req.Body)
		capturedBody = string(body)
	}).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"session_id": "session-1",
			"status": "closed",
			"message": {
				"role": "assistant",
				"content": "Dockerfile written.",
				"tool_calls": []
			}
		}`)),
	}, nil)

	response, err := service.ContinueSession(
		"org-1",
		"app-1",
		"session-1",
		[]DockerfileToolResult{
			{
				ToolCallID: "call-1",
				Output: map[string]any{
					"path":          "Dockerfile",
					"bytes_written": 42,
				},
			},
		},
	)

	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"sessionId": "session-1",
		"results": [{
			"tool_call_id": "call-1",
			"output": {
				"path": "Dockerfile",
				"bytes_written": 42
			}
		}]
	}`, capturedBody)
	assert.Equal(t, DockerfileSessionStatusClosed, response.Status)
	assert.Equal(t, "Dockerfile written.", response.Message.ContentOrEmpty())
	mockHTTPClient.AssertExpectations(t)
}

func TestDockerfileSessionService_LogsVerboseMessages(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &DockerfileSessionService{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	var logs []string
	service.SetVerboseCallback(func(message string) {
		logs = append(logs, message)
	})

	mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusCreated,
		Body: io.NopCloser(strings.NewReader(`{
			"session": {"id": "session-1", "status": "requires_tool_results"},
			"output": {
				"session_id": "session-1",
				"status": "requires_tool_results",
				"message": {
					"role": "assistant",
					"content": null,
					"tool_calls": []
				}
			}
		}`)),
	}, nil)

	_, err := service.StartSession("org-1", "app-1")

	assert.NoError(t, err)
	assert.Contains(t, logs[0], "Starting Dockerfile session: POST https://api.example.com/api/organizations/org-1/apps/app-1/dockerfile")
	assert.Contains(t, logs[1], "Received HTTP 201 from POST https://api.example.com/api/organizations/org-1/apps/app-1/dockerfile")
	assert.Contains(t, logs[2], "Started Dockerfile session session-1 with status requires_tool_results")
	assert.Contains(t, logs[3], "Session response: session=session-1 status=requires_tool_results")
}
