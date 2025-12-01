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

func (cs *CodeService) GenerateDocker(printer *cprint.CPrinter, cmd *cobra.Command) error {

	hasChanges, _ := gitutil.CheckForChangesNotOnRemote()
	if hasChanges {
		isOk := prompt.PromptYesNo(cmd, "You currently have changes that are not on the remote. Would you like to generate the dockerfile without these changes?", false)
		if !isOk.Confirmed {
			return fmt.Errorf("stopped")
		}
	}

	printer = cprint.NewCPrinter(flags.VerboseFlag)

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
		Run: hyphenRun,
	}

	statusDisplay := tea.NewProgram(modelThing)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		ioService := socketio.NewService()
		if err := ioService.Connect(orgId); err != nil {
			printer.Error(cmd, fmt.Errorf("failed to connect to Socket.io: %w", err))
			return
		}
		defer ioService.Disconnect()

		done := make(chan struct{})

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

			// Extract run data
			runDataRaw, ok := payload["run"].(map[string]any)
			if !ok {
				return
			}

			runJSON, err := json.Marshal(runDataRaw)
			if err != nil {
				return
			}

			var runObj run.Run
			if err := json.Unmarshal(runJSON, &runObj); err != nil {
				return
			}

			data := RunData{
				Action: action,
				RunId:  runId,
				Run:    runObj,
			}

			statusDisplay.Send(data)

			if data.Action == "update" && data.Run.Status == "succeeded" {
				var codeChanges run.CodeChangeRunData
				err := json.Unmarshal(data.Run.Data, &codeChanges)
				if err != nil {
					printer.Error(cmd, fmt.Errorf("error unmarshaling to CodeChangeRunData: %w", err))
					close(done)
					return
				}

				gitutil.ApplyDiffs(codeChanges.Changes)
				statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
				close(done)
			}
		})

		<-done
	}()

	statusDisplay.Run()
	wg.Wait()
	return nil
}
