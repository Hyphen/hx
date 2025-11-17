package deploy

import (
	"fmt"
	"time"

	"github.com/Hyphen/cli/internal/build"
	Deployment "github.com/Hyphen/cli/internal/deployment"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
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

		statusModel := &Deployment.StatusModel{
			OrganizationId: orgId,
			DeploymentId:   selectedDeployment.ID,
			RunId:          run.ID,
			Pipeline:       models.DeploymentPipeline{}, // Start with empty pipeline to show loading message
			Service:        *service,
			AppUrl:         fmt.Sprintf("%s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), orgId, selectedDeployment.ID, run.ID),
		}
		statusDisplay := tea.NewProgram(statusModel, tea.WithAltScreen())

		finalStatus := make(chan string, 1)

		go func() {
			const pollInterval = 500 * time.Millisecond
			const maxPollDuration = 30 * time.Minute

			// Fetch and send initial pipeline data immediately
			// Do this in the goroutine to avoid blocking the TUI from rendering
			initialRun, err := service.GetDeploymentRun(orgId, selectedDeployment.ID, run.ID)
			if err == nil {
				statusDisplay.Send(Deployment.RunMessageData{
					Type:     "run",
					Status:   initialRun.Status,
					RunId:    initialRun.ID,
					Id:       initialRun.ID,
					Pipeline: &initialRun.Pipeline,
				})
			}

			startTime := time.Now()

			ticker := time.NewTicker(pollInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if time.Since(startTime) > maxPollDuration {
						finalStatus <- "timeout"
						statusDisplay.Quit()
						return
					}

					deploymentRun, err := service.GetDeploymentRun(orgId, selectedDeployment.ID, run.ID)
					if err != nil {
						continue
					}

					// Send the updated pipeline in the message to avoid blocking the Update method
					statusDisplay.Send(Deployment.RunMessageData{
						Type:     "run",
						Status:   deploymentRun.Status,
						RunId:    deploymentRun.ID,
						Id:       deploymentRun.ID,
						Pipeline: &deploymentRun.Pipeline,
					})

					if deploymentRun.Status == "succeeded" || deploymentRun.Status == "failed" || deploymentRun.Status == "canceled" {
						finalStatus <- deploymentRun.Status
						statusDisplay.Quit()
						return
					}
				}
			}
		}()

		if _, err := statusDisplay.Run(); err != nil {
			printer.Error(cmd, err)
			return
		}

		// Show final status after TUI exits
		status := <-finalStatus
		switch status {
		case "succeeded":
			printer.GreenPrint(fmt.Sprintf("✓ Deployment succeeded: %s", statusModel.AppUrl))
		case "failed":
			printer.Error(cmd, fmt.Errorf("✗ Deployment failed: %s", statusModel.AppUrl))
		case "canceled":
			printer.YellowPrint(fmt.Sprintf("✗ Deployment canceled: %s", statusModel.AppUrl))
		case "timeout":
			printer.Error(cmd, fmt.Errorf("✗ Deployment timed out after 30 minutes"))
		}

	},
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
