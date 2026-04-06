package code

import (
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestRunDockerfileSession_ExecutesToolCallsUntilClosed(t *testing.T) {
	service := NewService()
	sessionClient := &fakeDockerfileSessionClient{
		startResponse: &DockerfileSessionCreateResponse{
			Session: DockerfileSession{
				ID: "session-1",
			},
			Output: DockerfileTurnResponse{
				SessionID: "session-1",
				Status:    DockerfileSessionStatusRequiresToolResults,
				Message: DockerfileAssistantMessage{
					ToolCalls: []DockerfileToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: DockerfileToolFunction{
								Name:      "list_files",
								Arguments: `{"path":"."}`,
							},
						},
					},
				},
			},
		},
		continueResponses: []*DockerfileTurnResponse{
			{
				SessionID: "session-1",
				Status:    DockerfileSessionStatusClosed,
				Message: DockerfileAssistantMessage{
					Content: stringPtr("Dockerfile written."),
				},
			},
		},
	}
	toolRunner := &fakeFilesystemToolRunner{
		results: []DockerfileToolResult{
			{
				ToolCallID: "call-1",
				Output: map[string]any{
					"entries": []map[string]string{{"path": "package.json", "type": "file"}},
				},
			},
		},
	}

	var statuses []string
	var successSummary string

	err := service.runDockerfileSession(
		sessionClient,
		toolRunner,
		"org-1",
		"app-1",
		dockerGenCallbacks{
			onStatus: func(status string) {
				statuses = append(statuses, status)
			},
			onSuccess: func(summary string) {
				successSummary = summary
			},
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, []string{"Starting Dockerfile generation", "Inspecting repository structure"}, statuses)
	assert.Equal(t, "Dockerfile written.", successSummary)
	assert.Len(t, toolRunner.calls, 1)
	assert.Equal(t, "list_files", toolRunner.calls[0][0].Function.Name)
	assert.Len(t, sessionClient.continueCalls, 1)
	assert.Equal(t, "session-1", sessionClient.continueCalls[0].sessionID)
	assert.Equal(t, "call-1", sessionClient.continueCalls[0].results[0].ToolCallID)
}

func TestRunDockerfileSession_FailsIfSessionDoesNotClose(t *testing.T) {
	service := NewService()
	sessionClient := &fakeDockerfileSessionClient{
		startResponse: &DockerfileSessionCreateResponse{
			Session: DockerfileSession{
				ID: "session-1",
			},
			Output: DockerfileTurnResponse{
				SessionID: "session-1",
				Status:    DockerfileSessionStatusReady,
				Message: DockerfileAssistantMessage{
					Content: stringPtr("I need more input."),
				},
			},
		},
	}

	var callbackErr error
	err := service.runDockerfileSession(
		sessionClient,
		&fakeFilesystemToolRunner{},
		"org-1",
		"app-1",
		dockerGenCallbacks{
			onError: func(runErr error) {
				callbackErr = runErr
			},
		},
	)

	assert.ErrorContains(t, err, "dockerfile generation stopped before the session closed")
	assert.ErrorContains(t, callbackErr, "dockerfile generation stopped before the session closed")
}

func TestEnsurePrinterReturnsDefaultWhenNil(t *testing.T) {
	assert.NotNil(t, ensurePrinter(nil))
}

func TestEnsurePrinterReturnsProvidedPrinter(t *testing.T) {
	printer := ensurePrinter(nil)
	assert.Same(t, printer, ensurePrinter(printer))
}

func TestConfiguredAppIDReturnsErrorWhenMissing(t *testing.T) {
	_, err := configuredAppID(config.Config{})
	assert.EqualError(t, err, "No app ID provided and no default found in manifest")
}

func TestConfiguredAppIDReturnsValueWhenPresent(t *testing.T) {
	appID := "app_123"
	value, err := configuredAppID(config.Config{AppId: &appID})
	assert.NoError(t, err)
	assert.Equal(t, appID, value)
}

type fakeDockerfileSessionClient struct {
	startResponse     *DockerfileSessionCreateResponse
	startErr          error
	continueResponses []*DockerfileTurnResponse
	continueErr       error
	continueCalls     []fakeContinueCall
}

type fakeContinueCall struct {
	sessionID string
	results   []DockerfileToolResult
}

func (f *fakeDockerfileSessionClient) StartSession(_, _ string) (*DockerfileSessionCreateResponse, error) {
	return f.startResponse, f.startErr
}

func (f *fakeDockerfileSessionClient) ContinueSession(
	_, _, sessionID string,
	results []DockerfileToolResult,
) (*DockerfileTurnResponse, error) {
	f.continueCalls = append(f.continueCalls, fakeContinueCall{
		sessionID: sessionID,
		results:   results,
	})
	if f.continueErr != nil {
		return nil, f.continueErr
	}
	if len(f.continueResponses) == 0 {
		return nil, assert.AnError
	}

	response := f.continueResponses[0]
	f.continueResponses = f.continueResponses[1:]
	return response, nil
}

type fakeFilesystemToolRunner struct {
	results []DockerfileToolResult
	calls   [][]DockerfileToolCall
}

func (f *fakeFilesystemToolRunner) ExecuteToolCalls(
	toolCalls []DockerfileToolCall,
	_ func(string),
) []DockerfileToolResult {
	f.calls = append(f.calls, toolCalls)
	return f.results
}

func stringPtr(value string) *string {
	return &value
}
