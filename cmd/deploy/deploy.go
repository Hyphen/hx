package deploy

import (
	"encoding/json"
	"fmt"

	"github.com/Hyphen/cli/internal/build"
	Deployment "github.com/Hyphen/cli/internal/deployment"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/Hyphen/cli/pkg/prompt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	noBuild bool
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

Use 'hyphen deploy --help' for more information about available flags.
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

		appSources := []Deployment.AppSources{
			// {
			// 	AppId: "app_67af84d8cf5902a8f372bbcc",
			// 	Artifact: Deployment.Artifact{
			// 		Type: "Docker",
			// 		Image: Deployment.Image{
			// 			//URI: "us-docker.pkg.dev/hyphenai/public/deploy-demo",
			// 			URI: "deploydevelopmentregistry.azurecr.io/wopr:0f3f7c3",
			// 		},
			// 	},
			// },
		}

		if noBuild {
			// check for build ID
		} else {
			service := build.NewService()
			result, err := service.RunBuild(printer, flags.VerboseFlag)
			if err != nil {
				printer.Error(cmd, err)
				return
			}
			appSources = append(appSources, Deployment.AppSources{
				AppId:    result.App.ID,
				Artifact: result.Artifact,
			},
			)
		}

		printer.Print(fmt.Sprintf("Triggering a run of: %s", selectedDeployment.Name))

		run, err := service.CreateRun(orgId, selectedDeployment.ID, appSources)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to create run: %w", err))
			return
		}
		printer.Print(fmt.Sprintf("Run Details: %s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), orgId, selectedDeployment.ID, run.ID))

		statusModel := Deployment.StatusModel{
			OrganizationId: orgId,
			DeploymentId:   selectedDeployment.ID,
			RunId:          run.ID,
			Pipeline:       run.Pipeline,
			Service:        *service,
		}
		statusDisplay := tea.NewProgram(statusModel)

		go func() {
			client := httputil.NewHyphenHTTPClient()
			conn, err := client.GetWebsocketConnection("ws://localhost:4000/api/websockets/eventStream")
			if err != nil {
				printer.Error(cmd, fmt.Errorf("failed to connect to WebSocket: %w", err))
				return
			}
			defer conn.Close()
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

				var wsMessage Deployment.WebSocketMessage
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
					var data Deployment.LogMessageData
					err = json.Unmarshal(wsMessage.Data, &data)
					if err != nil {
						continue
					}

					// TODO: this should go in a second pane or something
					// timestamp := time.UnixMilli(data.Timestamp)
					// formattedTime := timestamp.Format("15:04:05")
					// log := fmt.Sprintf("[%s] %s: %s", formattedTime, data.Level, data.Message)
					// printer.PrintVerbose(log)
				} else if _, ok := typeCheck["type"]; ok {
					var data Deployment.RunMessageData
					err = json.Unmarshal(wsMessage.Data, &data)
					if err != nil {
						continue
					}

					statusDisplay.Send(data)

					if data.Type == "run" && (data.Status == "succeeded" || data.Status == "failed" || data.Status == "canceled") {
						statusDisplay.Quit()
						break messageListener
					}
				}
			}
		}()
		statusDisplay.Run()

	},
}

func FindStepOrTaskByID(pipeline Deployment.Pipeline, id string) (interface{}, bool) {
	// Helper function to recursively search steps
	var searchSteps func(steps []Deployment.Step) (interface{}, bool)
	searchSteps = func(steps []Deployment.Step) (interface{}, bool) {
		for _, step := range steps {
			// Check if the current step matches the ID
			if step.ID == id {
				return step, true
			}

			// Check tasks within the step
			for _, task := range step.Tasks {
				if task.ID == id {
					return task, true
				}
			}

			// Recursively search in parallel steps
			if result, found := searchSteps(step.ParallelSteps); found {
				return result, true
			}
		}
		return nil, false
	}

	// Start searching from the top-level steps
	return searchSteps(pipeline.Steps)
}

func init() {
	DeployCmd.Flags().BoolVar(&noBuild, "no-build", false, "Skip the build step")
}
