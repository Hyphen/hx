package deploy

import (
	"fmt"
	"sync"

	"github.com/Hyphen/cli/internal/build"
	Deployment "github.com/Hyphen/cli/internal/deployment"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/Hyphen/cli/pkg/socketio"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			return err
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
			return fmt.Errorf("failed to list apps: %w", err)
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
				return err
			}

			if choice.Id == "" {
				printer.YellowPrint(("no choice made, canceling deploy"))
				return nil
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
			result, err := service.RunBuild(printer, firstApp.DeploymentSettings.ProjectEnvironment.ID, flags.VerboseFlag, flags.DockerfileFlag)
			if err != nil {
				return err
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
			return fmt.Errorf("failed to create run: %w", err)
		}

		statusModel := Deployment.StatusModel{
			OrganizationId: orgId,
			DeploymentId:   selectedDeployment.ID,
			RunId:          run.ID,
			Pipeline:       run.Pipeline,
			Service:        *service,
			AppUrl:         fmt.Sprintf("%s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), orgId, selectedDeployment.ID, run.ID),
		}
		statusDisplay := tea.NewProgram(statusModel)

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			ioService := socketio.NewService()
			if err := ioService.Connect(orgId); err != nil {
				printer.Error(cmd, fmt.Errorf("failed to connect to Socket.io: %w", err))
				wg.Done()
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
				if runId != run.ID {
					return
				}

				data, ok := payload["data"].(map[string]any)
				if !ok {
					return
				}

				runStatus, _ := data["status"].(string)
				if runStatus != "" {
					statusDisplay.Send(Deployment.RunMessageData{
						Type:   "run",
						RunId:  runId,
						Id:     runId,
						Status: runStatus,
					})

					if runStatus == "succeeded" || runStatus == "failed" || runStatus == "canceled" {
						statusDisplay.Quit()
						close(done)
						return
					}
				}

				if pipelineData, ok := data["pipeline"].(map[string]any); ok {
					extractStatusUpdates(pipelineData, runId, statusDisplay)
				}
			})

			ioService.Emit("Stream:RunLog:Start", map[string]any{
				"runId": run.ID,
			})

			<-done

			ioService.Emit("Stream:RunLog:Stop", map[string]any{
				"runId": run.ID,
			})

			wg.Done()
		}()

		statusDisplay.Run()
		wg.Wait()

		return nil
	},
}

func extractStatusUpdates(data map[string]any, runId string, statusDisplay *tea.Program) {
	if steps, ok := data["steps"].([]any); ok {
		for _, stepRaw := range steps {
			step, ok := stepRaw.(map[string]any)
			if !ok {
				continue
			}

			stepId, _ := step["id"].(string)
			stepStatus, _ := step["status"].(string)

			if stepId != "" && stepStatus != "" {
				statusDisplay.Send(Deployment.RunMessageData{
					Type:   "step",
					RunId:  runId,
					Id:     stepId,
					Status: stepStatus,
				})
			}

			if tasks, ok := step["tasks"].([]any); ok {
				for _, taskRaw := range tasks {
					task, ok := taskRaw.(map[string]any)
					if !ok {
						continue
					}

					taskId, _ := task["id"].(string)
					taskStatus, _ := task["status"].(string)

					if taskId != "" && taskStatus != "" {
						statusDisplay.Send(Deployment.RunMessageData{
							Type:   "task",
							RunId:  runId,
							Id:     taskId,
							Status: taskStatus,
						})
					}
				}
			}

			if parallelSteps, ok := step["parallelSteps"].([]any); ok {
				for _, psRaw := range parallelSteps {
					ps, ok := psRaw.(map[string]any)
					if !ok {
						continue
					}
					extractStatusUpdates(map[string]any{"steps": []any{ps}}, runId, statusDisplay)
				}
			}
		}
	}
}

func FindStepOrTaskByID(pipeline models.DeploymentPipeline, id string) (interface{}, bool) {
	var searchSteps func(steps []models.DeploymentStep) (interface{}, bool)
	searchSteps = func(steps []models.DeploymentStep) (interface{}, bool) {
		for _, step := range steps {
			if step.ID == id {
				return step, true
			}

			for _, task := range step.Tasks {
				if task.ID == id {
					return task, true
				}
			}

			if result, found := searchSteps(step.ParallelSteps); found {
				return result, true
			}
		}
		return nil, false
	}

	return searchSteps(pipeline.Steps)
}

func init() {
	DeployCmd.Flags().BoolVar(&noBuild, "no-build", false, "Skip the build step")
	DeployCmd.Flags().StringVarP(&flags.DockerfileFlag, "dockerfile", "f", "", "Path to Dockerfile (e.g., ./Dockerfile or ./docker/Dockerfile.prod)")
}
