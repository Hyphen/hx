package code

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
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
	var successResult dockerGenSuccess

	err := service.runDockerfileSession(
		context.Background(),
		sessionClient,
		toolRunner,
		"org-1",
		"app-1",
		dockerGenCallbacks{
			onStatus: func(status string) {
				statuses = append(statuses, status)
			},
			onSuccess: func(result dockerGenSuccess) {
				successResult = result
			},
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, []string{"Starting Dockerfile generation", "Inspecting repository structure"}, statuses)
	assert.Equal(t, dockerGenSuccess{Summary: "Dockerfile written.", FilesWritten: true}, successResult)
	assert.Len(t, toolRunner.calls, 1)
	assert.Equal(t, "list_files", toolRunner.calls[0][0].Function.Name)
	assert.Len(t, sessionClient.continueCalls, 1)
	assert.Equal(t, "session-1", sessionClient.continueCalls[0].sessionID)
	assert.Equal(t, "call-1", sessionClient.continueCalls[0].results[0].ToolCallID)
}

func TestRunDockerfileSession_TreatsReadyWithSummaryAsSuccess(t *testing.T) {
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
					Content: stringPtr("The existing Dockerfile and .dockerignore are already production-ready. No changes were necessary."),
				},
			},
		},
	}

	var successResult dockerGenSuccess
	err := service.runDockerfileSession(
		context.Background(),
		sessionClient,
		&fakeFilesystemToolRunner{},
		"org-1",
		"app-1",
		dockerGenCallbacks{
			onSuccess: func(result dockerGenSuccess) {
				successResult = result
			},
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, dockerGenSuccess{
		Summary:      "The existing Dockerfile and .dockerignore are already production-ready. No changes were necessary.",
		FilesWritten: false,
	}, successResult)
}

func TestRunDockerfileSession_FailsIfReadyResponseHasNoSummary(t *testing.T) {
	service := NewService()
	sessionClient := &fakeDockerfileSessionClient{
		startResponse: &DockerfileSessionCreateResponse{
			Session: DockerfileSession{
				ID: "session-1",
			},
			Output: DockerfileTurnResponse{
				SessionID: "session-1",
				Status:    DockerfileSessionStatusReady,
				Message:   DockerfileAssistantMessage{},
			},
		},
	}

	var callbackErr error
	err := service.runDockerfileSession(
		context.Background(),
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

func TestRunDockerfileSession_FailsIfStartSessionReturnsNilResponse(t *testing.T) {
	service := NewService()
	err := service.runDockerfileSession(
		context.Background(),
		&fakeDockerfileSessionClient{},
		&fakeFilesystemToolRunner{},
		"org-1",
		"app-1",
		dockerGenCallbacks{},
	)

	assert.EqualError(t, err, "dockerfile session did not return a response")
}

func TestRunDockerfileSession_StopsWhenContextIsCanceled(t *testing.T) {
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
	}

	ctx, cancel := context.WithCancel(context.Background())
	toolRunner := &fakeFilesystemToolRunner{
		onExecute: cancel,
		results: []DockerfileToolResult{
			{
				ToolCallID: "call-1",
			},
		},
	}

	err := service.runDockerfileSession(
		ctx,
		sessionClient,
		toolRunner,
		"org-1",
		"app-1",
		dockerGenCallbacks{},
	)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, sessionClient.continueCalls)
}

func TestGenerateDocker_ReturnsErrorForMonorepos(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")

	assert.NoError(t, os.MkdirAll(homeDir, 0o755))
	t.Setenv("HOME", homeDir)

	previousDir, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})

	assert.NoError(t, os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("local-only change\n"), 0o644))

	isMonorepo := true
	appID := "app_123"
	assert.NoError(t, config.InitializeConfig(config.Config{
		AppId:      &appID,
		IsMonorepo: &isMonorepo,
	}, config.ManifestConfigFile))

	originalOrgFlag := flags.OrganizationFlag
	flags.OrganizationFlag = "org_123"
	t.Cleanup(func() {
		flags.OrganizationFlag = originalOrgFlag
	})

	cmd := &cobra.Command{Use: "docker"}
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("no", true, "")

	err = NewService().GenerateDocker(nil, cmd)

	assert.EqualError(t, err, "docker generation for monorepos is not supported yet")
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

func (f *fakeDockerfileSessionClient) StartSession(
	_ context.Context,
	_, _ string,
) (*DockerfileSessionCreateResponse, error) {
	return f.startResponse, f.startErr
}

func (f *fakeDockerfileSessionClient) ContinueSession(
	_ context.Context,
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
	results   []DockerfileToolResult
	calls     [][]DockerfileToolCall
	onExecute func()
}

func (f *fakeFilesystemToolRunner) ExecuteToolCalls(
	toolCalls []DockerfileToolCall,
	_ func(string),
) []DockerfileToolResult {
	f.calls = append(f.calls, toolCalls)
	if f.onExecute != nil {
		f.onExecute()
	}
	return f.results
}

func stringPtr(value string) *string {
	return &value
}
