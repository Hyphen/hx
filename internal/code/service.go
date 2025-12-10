package code

import (
	"encoding/json"
	"fmt"
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
)

type CodeService struct {
}

func NewService() *CodeService {
	return &CodeService{}
}

func (cs *CodeService) GenerateDocker(_ *cprint.CPrinter, cmd *cobra.Command) error {

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

	modelThing := GenerateDockerRunModel{
		Run:         hyphenRun,
		VerboseMode: flags.VerboseFlag,
	}

	statusDisplay := tea.NewProgram(modelThing)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		ioService := socketio.NewService()

		// Set up verbose callback to send messages to the TUI
		if flags.VerboseFlag {
			ioService.SetVerboseCallback(func(msg string) {
				statusDisplay.Send(VerboseMessage{Content: msg})
			})
		}

		if err := ioService.Connect(orgId); err != nil {
			statusDisplay.Send(ErrorMessage{Error: fmt.Errorf("failed to connect to Socket.io: %w", err)})
			wg.Done()
			return
		}
		defer ioService.Disconnect()

		done := make(chan struct{})
		// This semaphore is defensive in case we get another message after we're "done" to avoid panicing from closing the done channel twice
		var doneOnce sync.Once

		// Helper to apply changes and exit
		applyChangesAndExit := func(changes []run.DiffResult) {
			if flags.VerboseFlag {
				statusDisplay.Send(VerboseMessage{Content: fmt.Sprintf("Applying %d changes", len(changes))})
			}
			gitutil.ApplyDiffs(changes)
			statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
			doneOnce.Do(func() { close(done) })
		}

		// Helper to handle errors and exit
		handleErrorAndExit := func(err error) {
			statusDisplay.Send(ErrorMessage{Error: err})
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

				statusDisplay.Send(RunData{Action: action, RunId: runId, Run: runObj})

				if action == "update" && runObj.Status == "succeeded" {
					var codeChanges run.CodeChangeRunData
					if err := json.Unmarshal(runObj.Data, &codeChanges); err != nil {
						handleErrorAndExit(fmt.Errorf("error unmarshaling CodeChangeRunData: %w", err))
						return
					}
					applyChangesAndExit(codeChanges.Changes)
				}
			} else if dataRaw, ok := payload["data"].(map[string]any); ok {
				status, _ := dataRaw["status"].(string)

				if flags.VerboseFlag {
					statusDisplay.Send(VerboseMessage{Content: fmt.Sprintf("Event: action=%s, status=%s", action, status)})
				}

				if action == "update" && status == "succeeded" {
					innerData, ok := dataRaw["data"].(map[string]any)
					if !ok {
						if flags.VerboseFlag {
							statusDisplay.Send(VerboseMessage{Content: "No inner 'data' field, waiting..."})
						}
						return
					}

					changesRaw, ok := innerData["changes"].([]any)
					if !ok || len(changesRaw) == 0 {
						if flags.VerboseFlag {
							statusDisplay.Send(VerboseMessage{Content: "No changes in inner data, waiting..."})
						}
						return
					}

					changesJSON, err := json.Marshal(changesRaw)
					if err != nil {
						handleErrorAndExit(fmt.Errorf("error marshaling changes: %w", err))
						return
					}

					var changes []run.DiffResult
					if err := json.Unmarshal(changesJSON, &changes); err != nil {
						handleErrorAndExit(fmt.Errorf("error unmarshaling changes: %w", err))
						return
					}

					applyChangesAndExit(changes)
				}
			}
		})

		<-done
		wg.Done()
	}()

	statusDisplay.Run()
	wg.Wait()
	return nil
}
