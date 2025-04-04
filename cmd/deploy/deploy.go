package deploy

import (
	"encoding/json"
	"fmt"
	"time"

	Deployment "github.com/Hyphen/cli/internal/deployment"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var (
	printer *cprint.CPrinter
)

var DeployCmd = &cobra.Command{
	Use:   "deploy <deployment>",
	Short: "Run a deployment",
	Long: `
The deploy command runs a deployment for a given deployment name.

Usage:
	hyphen deploy <deployment> [flags]

Examples:
hyphen deploy deploy-dev

Use 'hyphen link --help' for more information about available flags.
`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		deploymentName := ""
		if len(args) > 0 {
			deploymentName = args[0]
		}

		printer = cprint.NewCPrinter(flags.VerboseFlag)

		service := Deployment.NewService()

		// TODO: I'm not sure that proceeding if we find just one is right
		// I can see wanting to always prompt but for now let's just proceed
		deployments, err := service.SearchDeployments(orgId, deploymentName, 50, 1)

		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to list apps: %w", err))
			return
		}

		selectedDeployment := deployments[0]

		if len(deployments) > 1 {
			choices := make([]prompt.Choice, len(deployments))
			for i, deployment := range deployments {
				choices[i] = prompt.Choice{
					Id:      deployment.ID,
					Display: fmt.Sprintf("%s (%s)", deployment.Name, deployment.ID),
				}
			}

			choice, err := prompt.PromptSelection(choices, "Select a deployment to run:")

			if err != nil {
				printer.Error(cmd, err)
				return
			}

			if choice.Id == "" {
				printer.YellowPrint(("no choice made, canceling deploy"))
				return
			}

			for _, deployment := range deployments {
				if deployment.ID == choice.Id {
					selectedDeployment = deployment
					break
				}
			}
		}

		printer.Print(fmt.Sprintf("Triggering a run of: %s", selectedDeployment.Name))

		run, err := service.CreateRun(orgId, selectedDeployment.ID)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to create run: %w", err))
			return
		}
		printer.Print(fmt.Sprintf("Run Details: %s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), orgId, selectedDeployment.ID, run.ID))

		client := httputil.NewHyphenHTTPClient()
		conn, err := client.GetWebsocketConnection("ws://localhost:4000/api/websockets/eventStream")
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to connect to WebSocket: %w", err))
			return
		}
		defer conn.Close()

		printer.Info("Streaming logs...")

		conn.WriteJSON(
			map[string]interface{}{
				"eventStreamTopic": "deploymentRun",
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

			if _, ok := typeCheck["level"]; ok {
				var data LogMessageData
				err = json.Unmarshal(wsMessage.Data, &data)
				if err != nil {
					continue
				}

				timestamp := time.UnixMilli(data.Timestamp)
				formattedTime := timestamp.Format("15:04:05")
				log := fmt.Sprintf("[%s] %s: %s", formattedTime, data.Level, data.Message)
				switch data.Level {
				case "info":
					printer.Info(log)
				case "warn":
					printer.Warning(log)
				case "error":
					printer.YellowPrint(log)
				default:
					printer.Print(log)
				}
			} else if _, ok := typeCheck["type"]; ok {
				var data RunMessageData
				err = json.Unmarshal(wsMessage.Data, &data)
				if err != nil {
					continue
				}
				switch data.Type {
				case "run":
					printer.Print(fmt.Sprintf("[👟] Run %s", data.Status))
					if data.Status == "succeeded" {
						break messageListener
					}
				case "step":
					printer.Print(fmt.Sprintf("[🪜] Step %s", data.Status))
				case "task":
					printer.Print(fmt.Sprintf("[📃] Task %s", data.Status))
				default:
					// ignore unknown types
				}
			}
		}
	},
}

type WebSocketMessage struct {
	EventStreamTopic string          `json:"eventStreamTopic"`
	OrganizationId   string          `json:"organizationId"`
	Data             json.RawMessage `json:"data"`
}

// Define the first data type (current structure)
type LogMessageData struct {
	Level        string   `json:"level"`
	Message      string   `json:"message"`
	RunId        string   `json:"runId"`
	Timestamp    int64    `json:"timestamp"`
	Id           string   `json:"id"`
	Parents      []string `json:"parents"`
	Organization struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"organization"`
}

// Define the second data type (new structure)
type RunMessageData struct {
	Type   string `json:"type"`
	RunId  string `json:"RunId"`
	Id     string `json:"id"`
	Status string `json:"status"`
}
