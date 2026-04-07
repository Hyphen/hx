package code

import (
	"fmt"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const maxDockerfileSessionTurns = 24

type CodeService struct{}

type dockerGenSuccess struct {
	Summary      string
	FilesWritten bool
}

type dockerGenCallbacks struct {
	onVerbose func(msg string)
	onStatus  func(status string)
	onSuccess func(result dockerGenSuccess)
	onError   func(err error)
}

type filesystemToolRunner interface {
	ExecuteToolCalls(toolCalls []DockerfileToolCall, onVerbose func(string)) []DockerfileToolResult
}

type verboseDockerfileSessionClient interface {
	SetVerboseCallback(func(string))
}

func NewService() *CodeService {
	return &CodeService{}
}

func (cs *CodeService) GenerateDocker(printer *cprint.CPrinter, cmd *cobra.Command) error {
	printer = ensurePrinter(printer)

	cfg, err := config.RestoreConfig()
	if err != nil {
		return err
	}

	orgID, err := flags.GetOrganizationID()
	if err != nil {
		return err
	}

	if cfg.IsMonorepoProject() {
		return fmt.Errorf("docker generation for monorepos is not supported yet")
	}

	appID, err := configuredAppID(cfg)
	if err != nil {
		return err
	}

	workspaceRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine workspace root: %w", err)
	}

	if shouldUseTUI() {
		return cs.generateDockerWithTUI(orgID, appID, workspaceRoot)
	}

	return cs.generateDockerWithoutTUI(printer, orgID, appID, workspaceRoot)
}

func shouldUseTUI() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func (cs *CodeService) runDockerfileSession(
	sessionClient DockerfileSessionClient,
	toolRunner filesystemToolRunner,
	orgID, appID string,
	callbacks dockerGenCallbacks,
) error {
	if verboseClient, ok := sessionClient.(verboseDockerfileSessionClient); ok {
		verboseClient.SetVerboseCallback(callbacks.onVerbose)
	}

	if callbacks.onStatus != nil {
		callbacks.onStatus("Starting Dockerfile generation")
	}

	createResponse, err := sessionClient.StartSession(orgID, appID)
	if err != nil {
		return failDockerGen(err, callbacks)
	}

	if createResponse.Session.ID == "" {
		return failDockerGen(fmt.Errorf("dockerfile session did not return a session id"), callbacks)
	}

	turn := createResponse.Output
	for turnIndex := 0; turnIndex < maxDockerfileSessionTurns; turnIndex++ {
		switch turn.Status {
		case DockerfileSessionStatusRequiresToolResults:
			if len(turn.Message.ToolCalls) == 0 {
				return failDockerGen(fmt.Errorf("dockerfile session requested tool results without any tool calls"), callbacks)
			}

			if callbacks.onVerbose != nil {
				assistantMessage := strings.TrimSpace(turn.Message.ContentOrEmpty())
				if assistantMessage != "" {
					callbacks.onVerbose(fmt.Sprintf("Assistant: %s", assistantMessage))
				}
			}

			if callbacks.onStatus != nil {
				callbacks.onStatus(describeToolCalls(turn.Message.ToolCalls))
			}

			results := toolRunner.ExecuteToolCalls(turn.Message.ToolCalls, callbacks.onVerbose)
			nextTurn, continueErr := sessionClient.ContinueSession(orgID, appID, createResponse.Session.ID, results)
			if continueErr != nil {
				return failDockerGen(continueErr, callbacks)
			}
			turn = *nextTurn
		case DockerfileSessionStatusClosed:
			if callbacks.onSuccess != nil {
				callbacks.onSuccess(dockerGenSuccess{
					Summary:      strings.TrimSpace(turn.Message.ContentOrEmpty()),
					FilesWritten: true,
				})
			}
			return nil
		case DockerfileSessionStatusFailed:
			return failDockerGen(
				fmt.Errorf("dockerfile generation failed: %s", sessionMessage(turn.Message.ContentOrEmpty(), "no details provided")),
				callbacks,
			)
		case DockerfileSessionStatusReady:
			readySummary := strings.TrimSpace(turn.Message.ContentOrEmpty())
			if readySummary != "" {
				if callbacks.onSuccess != nil {
					callbacks.onSuccess(dockerGenSuccess{
						Summary:      readySummary,
						FilesWritten: false,
					})
				}
				return nil
			}
			return failDockerGen(
				fmt.Errorf(
					"dockerfile generation stopped before the session closed: %s",
					sessionMessage(readySummary, "no details provided"),
				),
				callbacks,
			)
		default:
			return failDockerGen(fmt.Errorf("unexpected dockerfile session status: %s", turn.Status), callbacks)
		}
	}

	return failDockerGen(
		fmt.Errorf("dockerfile generation exceeded %d session turns", maxDockerfileSessionTurns),
		callbacks,
	)
}

func (cs *CodeService) generateDockerWithTUI(orgID, appID, workspaceRoot string) error {
	toolRunner, err := NewFilesystemToolExecutor(workspaceRoot)
	if err != nil {
		return err
	}

	statusDisplay := tea.NewProgram(GenerateDockerSessionModel{
		Status:      "Starting Dockerfile generation",
		VerboseMode: flags.VerboseFlag,
	})

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- cs.runDockerfileSession(
			NewDockerfileSessionService(),
			toolRunner,
			orgID,
			appID,
			dockerGenCallbacks{
				onVerbose: func(msg string) {
					statusDisplay.Send(VerboseMessage{Content: msg})
				},
				onStatus: func(status string) {
					statusDisplay.Send(StatusMessage{Content: status})
				},
				onSuccess: func(result dockerGenSuccess) {
					statusDisplay.Send(SuccessMessage{
						Summary:      result.Summary,
						FilesWritten: result.FilesWritten,
					})
				},
				onError: func(err error) {
					statusDisplay.Send(ErrorMessage{Error: err})
				},
			},
		)
	}()

	_, tuiErr := statusDisplay.Run()
	runErr := <-runErrCh
	if tuiErr != nil {
		return tuiErr
	}
	return runErr
}

func (cs *CodeService) generateDockerWithoutTUI(printer *cprint.CPrinter, orgID, appID, workspaceRoot string) error {
	toolRunner, err := NewFilesystemToolExecutor(workspaceRoot)
	if err != nil {
		return err
	}

	printer.Print("Generating Dockerfile (this may take a few seconds)...")

	return cs.runDockerfileSession(
		NewDockerfileSessionService(),
		toolRunner,
		orgID,
		appID,
		dockerGenCallbacks{
			onVerbose: func(msg string) {
				printer.Print(fmt.Sprintf("  [verbose] %s", msg))
			},
			onStatus: func(status string) {
				printer.Print(status)
			},
			onSuccess: func(result dockerGenSuccess) {
				if result.FilesWritten {
					printer.Success("Dockerfile and .dockerignore generated! You may choose to check them in if you like.")
				} else {
					printer.Success("Existing Dockerfile and .dockerignore already look good. No files were generated or changed.")
				}
				if result.Summary != "" {
					printer.Print(result.Summary)
				}
			},
			onError: nil,
		},
	)
}

func failDockerGen(err error, callbacks dockerGenCallbacks) error {
	if callbacks.onError != nil {
		callbacks.onError(err)
	}
	return err
}

func describeToolCalls(toolCalls []DockerfileToolCall) string {
	if len(toolCalls) == 0 {
		return "Waiting for Dockerfile generation"
	}

	hasWrite := false
	hasRead := false
	hasSearch := false
	hasList := false

	for _, toolCall := range toolCalls {
		switch toolCall.Function.Name {
		case "write_file":
			hasWrite = true
		case "read_file":
			hasRead = true
		case "search":
			hasSearch = true
		case "list_files":
			hasList = true
		}
	}

	switch {
	case hasWrite:
		return "Writing Dockerfile"
	case hasRead:
		return "Reading repository files"
	case hasSearch:
		return "Searching repository files"
	case hasList:
		return "Inspecting repository structure"
	default:
		return "Running filesystem tools"
	}
}

func sessionMessage(content, fallback string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func ensurePrinter(printer *cprint.CPrinter) *cprint.CPrinter {
	if printer != nil {
		return printer
	}
	return cprint.NewCPrinter(flags.VerboseFlag)
}

func configuredAppID(cfg config.Config) (string, error) {
	if cfg.AppId == nil || strings.TrimSpace(*cfg.AppId) == "" {
		return "", fmt.Errorf("No app ID provided and no default found in manifest")
	}

	return *cfg.AppId, nil
}
