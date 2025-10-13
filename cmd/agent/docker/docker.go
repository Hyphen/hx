package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/run"
	runp "github.com/Hyphen/cli/internal/run"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/httputil"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	printer *cprint.CPrinter
)

var DockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Generate a Dockerfile for the given application",
	Long:  `Generate a Dockerfile for the given application.`,
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		config, err := config.RestoreConfig()
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		orgId, err := flags.GetOrganizationID()
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		if config.IsMonorepoProject() {
			printer.Error(cmd, fmt.Errorf("docker generation for monorepos is not supported yet"))
			return
		}

		service := runp.NewService()

		targetBranch := "main"
		targetBranch, _ = gitutil.GetCurrentBranch()
		run, err := service.CreateDockerFileRun(orgId, *config.AppId, targetBranch)
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		modelThing := DockerModelThingy{
			Run: run,
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

				if data.RunId != run.ID {
					continue
				}

				statusDisplay.Send(data)

				if data.Action == "update" && data.Run.Status == "succeeded" {
					var codeChanges runp.CodeChangeRunData
					err := json.Unmarshal(data.Run.Data, &codeChanges)
					if err != nil {
						printer.Error(cmd, fmt.Errorf("error unmarshaling to CodeChangeRunData: %w", err))
						break messageListener
					}

					applyDiffs(codeChanges.Changes)
					statusDisplay.Send(tea.KeyMsg{Type: tea.KeyEsc})
					break messageListener
				}

			}
		}()
		statusDisplay.Run()
	},
}

func applyDiffs(diffs []run.DiffResult) {
	fs := fsutil.NewFileSystem()
	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// iterate over diffs
	for _, diff := range diffs {
		// This is a delete
		if diff.To == "" {
			fullPath := filepath.Join(currentDir, diff.From)
			err := fs.Remove(fullPath)
			if err != nil {
				fmt.Printf("Error removing file %s: %v\n", fullPath, err)
			}
			continue
		}

		// This is a create or modify
		var contents []byte
		for i, chunk := range diff.Chunks {
			fmt.Printf("Chunk %d - Type: '%s', Content length: %d\n", i, chunk.Type, len(chunk.Content))
			// TODO: handle deletes to files
			if chunk.Type != "delete" {
				contents = append(contents, []byte(chunk.Content)...)
				fmt.Printf("Added chunk %d to contents, total length now: %d\n", i, len(contents))
			}
		}
		fullPath := filepath.Join(currentDir, diff.To)

		err := fs.WriteFile(fullPath, contents, 0o644)
		if err != nil {
			fmt.Printf("Error writing file %s: %v\n", fullPath, err)
		} else {
			fmt.Printf("Successfully wrote %d bytes to %s\n", len(contents), fullPath)
		}
	}

}
