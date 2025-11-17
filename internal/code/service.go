package code

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/run"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/prompt"
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

	go func() {
		const pollInterval = 500 * time.Millisecond
		const maxPollDuration = 30 * time.Minute

		printer.PrintVerbose("Monitoring code generation status via polling")
		startTime := time.Now()

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if time.Since(startTime) > maxPollDuration {
					printer.Error(cmd, fmt.Errorf("code generation polling timeout after %v", maxPollDuration))
					statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
					return
				}

				currentRun, err := service.GetRun(orgId, *config.AppId, hyphenRun.ID)
				if err != nil {
					printer.PrintVerbose(fmt.Sprintf("Error polling run status: %v", err))
					continue
				}

				printer.PrintVerbose(fmt.Sprintf("Run status: %s", currentRun.Status))

				// Send status update to display
				statusDisplay.Send(RunData{
					Action: "update",
					RunId:  currentRun.ID,
					Run:    *currentRun,
				})

				if currentRun.Status == "succeeded" {
					var codeChanges run.CodeChangeRunData
					err := json.Unmarshal(currentRun.Data, &codeChanges)
					if err != nil {
						printer.Error(cmd, fmt.Errorf("error unmarshaling to CodeChangeRunData: %w", err))
						statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
						return
					}

					printer.PrintVerbose("Code generation completed successfully, applying diffs")
					gitutil.ApplyDiffs(codeChanges.Changes)
					statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
					return
				}

				if currentRun.Status == "failed" || currentRun.Status == "canceled" {
					printer.Error(cmd, fmt.Errorf("code generation %s", currentRun.Status))
					statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
					return
				}
			}
		}
	}()
	statusDisplay.Run()
	return nil
}
