package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Hyphen/cli/internal/build"
	Deployment "github.com/Hyphen/cli/internal/deployment"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/Hyphen/cli/pkg/prompt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
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

		appSources := []Deployment.AppSources{}

		if noBuild {
			// check for build ID
		} else {
			// TODO: right now we are only supporting building one app at a time
			// we'll need to come back and fix this SOON
			firstApp := selectedDeployment.Apps[0]

			service := build.NewService()
			result, err := service.RunBuild(printer, firstApp.DeploymentSettings.ProjectEnvironment.ID, flags.VerboseFlag)
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

		printer.Print(fmt.Sprintf("Running %s", selectedDeployment.Name))

		run, err := service.CreateRun(orgId, selectedDeployment.ID, appSources)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to create run: %w", err))
			return
		}

		appUrl := fmt.Sprintf("%s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), orgId, selectedDeployment.ID, run.ID)

		if shouldUseTUI() {
			runWithTUI(cmd, orgId, selectedDeployment.ID, run, appUrl, service)
		} else {
			runWithoutTUI(cmd, orgId, appUrl)
		}
	},
}

func shouldUseTUI() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func runWithTUI(cmd *cobra.Command, orgId, deploymentId string, run *models.DeploymentRun, appUrl string, service *Deployment.DeploymentService) {
	statusModel := Deployment.StatusModel{
		OrganizationId: orgId,
		DeploymentId:   deploymentId,
		RunId:          run.ID,
		Pipeline:       run.Pipeline,
		Service:        *service,
		AppUrl:         appUrl,
		VerboseMode:    flags.VerboseFlag,
	}
	statusDisplay := tea.NewProgram(statusModel)

	// Ticker to update waiting seconds
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			statusDisplay.Send(Deployment.WaitingTickMessage{})
		}
	}()

	go func() {
		client := httputil.NewHyphenHTTPClient()
		url := fmt.Sprintf("%s/api/websockets/eventStream", apiconf.GetBaseWebsocketUrl())
		conn, err := client.GetWebsocketConnection(url)
		if err != nil {
			statusDisplay.Send(Deployment.ErrorMessage{Error: fmt.Errorf("failed to connect to WebSocket: %w", err)})
			return
		}
		defer conn.Close()

		if flags.VerboseFlag {
			statusDisplay.Send(Deployment.VerboseMessage{Content: "WebSocket connected"})
		}

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
				statusDisplay.Send(Deployment.ErrorMessage{Error: fmt.Errorf("error reading WebSocket message: %w", err)})
				break
			}

			// TODO: For now, log every WebSocket message in verbose mode, but remove once everything is finally working
			if flags.VerboseFlag {
				statusDisplay.Send(Deployment.VerboseMessage{Content: fmt.Sprintf("WebSocket Message: %s", string(message))})
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
				if flags.VerboseFlag {
					statusDisplay.Send(Deployment.VerboseMessage{Content: fmt.Sprintf("Log [%s]: %s", data.Level, data.Message)})
				}
			} else if _, ok := typeCheck["type"]; ok {
				var data Deployment.RunMessageData
				err = json.Unmarshal(wsMessage.Data, &data)
				if err != nil {
					continue
				}

				if flags.VerboseFlag {
					statusDisplay.Send(Deployment.VerboseMessage{Content: fmt.Sprintf("RunMessage: Type=%s, Id=%s, Status=%s", data.Type, data.Id, data.Status)})
				}

				statusDisplay.Send(data)

				if data.Type == "run" && (data.Status == "succeeded" || data.Status == "failed" || data.Status == "canceled") {
					if flags.VerboseFlag {
						statusDisplay.Send(Deployment.VerboseMessage{Content: fmt.Sprintf("Deployment reached terminal status: %s", data.Status)})
					}
					statusDisplay.Quit()
					break messageListener
				}
			}
		}
	}()
	statusDisplay.Run()
}

func runWithoutTUI(cmd *cobra.Command, orgId string, appUrl string) {
	printer.Print(fmt.Sprintf("Deployment URL: %s", appUrl))
	printer.Print("Monitoring deployment progress...")

	client := httputil.NewHyphenHTTPClient()
	url := fmt.Sprintf("%s/api/websockets/eventStream", apiconf.GetBaseWebsocketUrl())

	conn, err := client.GetWebsocketConnection(url)
	if err != nil {
		printer.Error(cmd, fmt.Errorf("failed to connect to WebSocket: %w", err))
		return
	}
	defer conn.Close()

	conn.WriteJSON(
		map[string]any{
			"eventStreamTopic": "deploymentRun",
			"organizationId":   orgId,
		},
	)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			printer.Error(cmd, fmt.Errorf("error reading WebSocket message: %w", err))
			return
		}

		var wsMessage Deployment.WebSocketMessage
		err = json.Unmarshal(message, &wsMessage)
		if err != nil {
			continue
		}

		var typeCheck map[string]interface{}
		err = json.Unmarshal(wsMessage.Data, &typeCheck)
		if err != nil {
			continue
		}

		if _, ok := typeCheck["type"]; ok {
			var data Deployment.RunMessageData
			err = json.Unmarshal(wsMessage.Data, &data)
			if err != nil {
				continue
			}

			// Print status updates
			switch data.Type {
			case "step", "task":
				printer.Print(fmt.Sprintf("  %s: %s", data.Type, data.Status))
			case "run":
				printer.Print(fmt.Sprintf("Deployment: %s", data.Status))

				switch data.Status {
				case "succeeded":
					printer.Success("Deployment completed successfully")
					return
				case "failed":
					printer.Error(cmd, fmt.Errorf("deployment failed"))
					return
				case "canceled":
					printer.Warning("Deployment was canceled")
					return
				}
			}
		}
	}
}

func FindStepOrTaskByID(pipeline models.DeploymentPipeline, id string) (interface{}, bool) {
	// Helper function to recursively search steps
	var searchSteps func(steps []models.DeploymentStep) (interface{}, bool)
	searchSteps = func(steps []models.DeploymentStep) (interface{}, bool) {
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
