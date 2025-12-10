package code

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/run"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/Hyphen/cli/pkg/socketio"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CodeService struct {
}

func NewService() *CodeService {
	return &CodeService{}
}

func (cs *CodeService) GenerateDocker(printer *cprint.CPrinter, cmd *cobra.Command) error {

	hasChanges, _ := gitutil.CheckForChangesNotOnRemote()
	if hasChanges {
		isOk := prompt.PromptYesNo(cmd, "You currently have changes that are not on the remote. Would you like to generate the dockerfile without these changes?", false)
		if !isOk.Confirmed {
			return fmt.Errorf("stopped")
		}
	}

	config, err := config.RestoreConfig()
	if err != nil {
		return err
	}

	orgId, err := flags.GetOrganizationID()
	if err != nil {
		return err
	}

	if config.IsMonorepoProject() {
		return fmt.Errorf("docker generation for monorepos is not supported yet")
	}

	service := run.NewService()

	targetBranch := "main"
	targetBranch, _ = gitutil.GetCurrentBranch()
	hyphenRun, err := service.CreateDockerFileRun(orgId, *config.AppId, targetBranch)
	if err != nil {
		return err
	}

	if shouldUseTUI() {
		return cs.generateDockerWithTUI(orgId, hyphenRun)
	}
	return cs.generateDockerWithoutTUI(printer, orgId, hyphenRun)
}

func shouldUseTUI() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// dockerGenCallbacks provides callbacks for the docker generation event handler
type dockerGenCallbacks struct {
	onVerbose func(msg string)
	onStatus  func(status run.RunStatus)
	onSuccess func(changes []run.DiffResult)
	onError   func(err error)
}

func (cs *CodeService) streamDockerGenEvents(orgId string, hyphenRun *run.Run, callbacks dockerGenCallbacks) error {
	ioService := socketio.NewService()

	if flags.VerboseFlag && callbacks.onVerbose != nil {
		ioService.SetVerboseCallback(callbacks.onVerbose)
	}

	if err := ioService.Connect(orgId); err != nil {
		return fmt.Errorf("failed to connect to Socket.io: %w", err)
	}
	defer ioService.Disconnect()

	done := make(chan struct{})
	var doneOnce sync.Once
	var resultErr error

	closeWithError := func(err error) {
		resultErr = err
		if callbacks.onError != nil {
			callbacks.onError(err)
		}
		doneOnce.Do(func() { close(done) })
	}

	closeWithSuccess := func(changes []run.DiffResult) {
		if callbacks.onVerbose != nil && flags.VerboseFlag {
			callbacks.onVerbose(fmt.Sprintf("Applying %d changes", len(changes)))
		}
		gitutil.ApplyDiffs(changes)
		if callbacks.onSuccess != nil {
			callbacks.onSuccess(changes)
		}
		doneOnce.Do(func() { close(done) })
	}

	ioService.On("Event:Run", func(args ...any) {
		if len(args) == 0 {
			return
		}

		payload, ok := args[0].(map[string]any)
		if !ok {
			return
		}

		runId, _ := payload["runId"].(string)
		if runId != hyphenRun.ID {
			return
		}

		action, _ := payload["action"].(string)

		// Handle two possible event formats:
		// 1. Legacy format: payload["run"] contains a Run object with status and data
		// 2. New format: payload["data"] contains status and changes directly
		if runDataRaw, ok := payload["run"].(map[string]any); ok {
			runJSON, err := json.Marshal(runDataRaw)
			if err != nil {
				return
			}

			var runObj run.Run
			if err := json.Unmarshal(runJSON, &runObj); err != nil {
				return
			}

			if callbacks.onStatus != nil {
				callbacks.onStatus(runObj.Status)
			}

			if action == "update" && runObj.Status == "succeeded" {
				var codeChanges run.CodeChangeRunData
				if err := json.Unmarshal(runObj.Data, &codeChanges); err != nil {
					closeWithError(fmt.Errorf("error unmarshaling CodeChangeRunData: %w", err))
					return
				}
				closeWithSuccess(codeChanges.Changes)
				return
			}

			if runObj.Status == "failed" {
				closeWithError(fmt.Errorf("dockerfile generation failed"))
				return
			}

			if runObj.Status == "canceled" {
				closeWithError(fmt.Errorf("dockerfile generation was canceled"))
				return
			}
		} else if dataRaw, ok := payload["data"].(map[string]any); ok {
			status, _ := dataRaw["status"].(string)

			if flags.VerboseFlag && callbacks.onVerbose != nil {
				callbacks.onVerbose(fmt.Sprintf("Event: action=%s, status=%s", action, status))
			}

			if status == "failed" {
				closeWithError(fmt.Errorf("dockerfile generation failed"))
				return
			}

			if status == "canceled" {
				closeWithError(fmt.Errorf("dockerfile generation was canceled"))
				return
			}

			if action == "update" && status == "succeeded" {
				innerData, ok := dataRaw["data"].(map[string]any)
				if !ok {
					if flags.VerboseFlag && callbacks.onVerbose != nil {
						callbacks.onVerbose("No inner 'data' field, waiting...")
					}
					return
				}

				changesRaw, ok := innerData["changes"].([]any)
				if !ok || len(changesRaw) == 0 {
					if flags.VerboseFlag && callbacks.onVerbose != nil {
						callbacks.onVerbose("No changes in inner data, waiting...")
					}
					return
				}

				changesJSON, err := json.Marshal(changesRaw)
				if err != nil {
					closeWithError(fmt.Errorf("error marshaling changes: %w", err))
					return
				}

				var changes []run.DiffResult
				if err := json.Unmarshal(changesJSON, &changes); err != nil {
					closeWithError(fmt.Errorf("error unmarshaling changes: %w", err))
					return
				}

				closeWithSuccess(changes)
			}
		}
	})

	<-done
	return resultErr
}

func (cs *CodeService) generateDockerWithTUI(orgId string, hyphenRun *run.Run) error {
	modelThing := GenerateDockerRunModel{
		Run:         hyphenRun,
		VerboseMode: flags.VerboseFlag,
	}

	statusDisplay := tea.NewProgram(modelThing)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := cs.streamDockerGenEvents(orgId, hyphenRun, dockerGenCallbacks{
			onVerbose: func(msg string) {
				statusDisplay.Send(VerboseMessage{Content: msg})
			},
			onStatus: func(status run.RunStatus) {
				statusDisplay.Send(RunData{Run: run.Run{Status: status}})
			},
			onSuccess: func(changes []run.DiffResult) {
				statusDisplay.Quit()
			},
			onError: func(err error) {
				statusDisplay.Send(ErrorMessage{Error: err})
			},
		})

		if err != nil {
			statusDisplay.Send(ErrorMessage{Error: err})
		}
	}()

	statusDisplay.Run()
	wg.Wait()
	return nil
}

func (cs *CodeService) generateDockerWithoutTUI(printer *cprint.CPrinter, orgId string, hyphenRun *run.Run) error {
	printer.Print("Generating Dockerfile (this may take a few seconds)...")

	return cs.streamDockerGenEvents(orgId, hyphenRun, dockerGenCallbacks{
		onVerbose: func(msg string) {
			printer.Print(fmt.Sprintf("  [verbose] %s", msg))
		},
		onStatus: func(status run.RunStatus) {
			printer.Print(fmt.Sprintf("Status: %s", status))
		},
		onSuccess: func(changes []run.DiffResult) {
			printer.Success("Dockerfile generated! You may choose to check it in if you like.")
		},
		onError: nil, // errors returned directly
	})
}
