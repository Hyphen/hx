package code

import (
	"encoding/json"
	"fmt"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/run"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/httputil"
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
		client := httputil.NewHyphenHTTPClient()
		url := fmt.Sprintf("%s/api/websockets/eventStream", apiconf.GetBaseWebsocketUrl())
		conn, err := client.GetWebsocketConnection(url)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to connect to WebSocket: %w", err))
			return
		}
		defer conn.Close()
		conn.WriteJSON(
			map[string]interface{}{
				"eventStreamTopic": "run",
				"organizationId":   orgId,
			},
		)
	messageListener:
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				printer.Error(cmd, fmt.Errorf("error reading WebSocket message: %w", err))
				break
			}

			var wsMessage WebSocketMessage
			err = json.Unmarshal(message, &wsMessage)
			if err != nil {
				continue
			}

			var typeCheck map[string]interface{}
			err = json.Unmarshal(wsMessage.Data, &typeCheck)
			if err != nil {
				printer.Error(cmd, fmt.Errorf("error unmarshalling Data for type check: %w", err))
				continue
			}

			var data RunData

			err = json.Unmarshal(wsMessage.Data, &data)
			if err != nil {
				continue
			}

			if data.RunId != hyphenRun.ID {
				continue
			}

			statusDisplay.Send(data)

			if data.Action == "update" && data.Run.Status == "succeeded" {
				var codeChanges run.CodeChangeRunData
				err := json.Unmarshal(data.Run.Data, &codeChanges)
				if err != nil {
					printer.Error(cmd, fmt.Errorf("error unmarshaling to CodeChangeRunData: %w", err))
					break messageListener
				}

				gitutil.ApplyDiffs(codeChanges.Changes)
				statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
				break messageListener
			}

		}
	}()
	statusDisplay.Run()
	return nil
}
