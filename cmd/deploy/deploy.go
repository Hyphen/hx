package deploy

import (
	"fmt"
	"os"
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

		if !selectedDeployment.IsReady {
			printer.Print("❌ There are issues blocking this deployment from being run.")
			for _, issue := range selectedDeployment.ReadinessIssues {
				if issue.Cloud != "" {
					printer.Print(fmt.Sprintf("  • %s (%s)", issue.Error, issue.Cloud))
				} else {
					printer.Print(fmt.Sprintf("  • %s", issue.Error))
				}
			}
			return nil
		}

		appSources := []Deployment.AppSources{}

		if noBuild {
			// check for build ID
		} else {
			// TODO: right now we are only supporting building one app at a time
			// we'll need to come back and fix this SOON
			firstApp := selectedDeployment.Apps[0]

			service := build.NewService()
			result, err := service.RunBuild(cmd, printer, firstApp.DeploymentSettings.ProjectEnvironment.ID, flags.VerboseFlag, flags.DockerfileFlag, false)
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

		appUrl := fmt.Sprintf("%s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), orgId, selectedDeployment.ID, run.ID)

		if shouldUseTUI() {
			runWithTUI(orgId, selectedDeployment.ID, run, appUrl, service)
		} else {
			runWithoutTUI(orgId, selectedDeployment.ID, run, appUrl, service)
		}

		return nil
	},
}

func shouldUseTUI() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// deployCallbacks provides callbacks for the deployment event handler
type deployCallbacks struct {
	onVerbose        func(msg string)
	onStatus         func(runId, status string)
	onPipelineUpdate func(pipelineData map[string]any, runId string)
	onComplete       func(status string)
	onError          func(err error)
}

func streamDeployEvents(orgId string, runID string, callbacks deployCallbacks) error {
	ioService := socketio.NewService()

	if flags.VerboseFlag && callbacks.onVerbose != nil {
		ioService.SetVerboseCallback(callbacks.onVerbose)
	}

	if err := ioService.Connect(orgId); err != nil {
		return fmt.Errorf("failed to connect to Socket.io: %w", err)
	}
	defer ioService.Disconnect()

	done := make(chan struct{})
	var doneOnce sync.Once

	ioService.On("Event:Run", func(args ...any) {
		if len(args) == 0 {
			return
		}

		payload, ok := args[0].(map[string]any)
		if !ok {
			return
		}

		eventRunId, _ := payload["runId"].(string)
		if eventRunId != runID {
			return
		}

		data, ok := payload["data"].(map[string]any)
		if !ok {
			return
		}

		runStatus, _ := data["status"].(string)
		if runStatus != "" {
			if flags.VerboseFlag && callbacks.onVerbose != nil {
				callbacks.onVerbose(fmt.Sprintf("Run status updated to %s", runStatus))
			}

			if callbacks.onStatus != nil {
				callbacks.onStatus(eventRunId, runStatus)
			}

			if runStatus == "succeeded" || runStatus == "failed" || runStatus == "canceled" {
				if flags.VerboseFlag && callbacks.onVerbose != nil {
					callbacks.onVerbose(fmt.Sprintf("Deployment ended with status %s", runStatus))
				}
				if callbacks.onComplete != nil {
					callbacks.onComplete(runStatus)
				}
				doneOnce.Do(func() { close(done) })
				return
			}
		}

		if pipelineData, ok := data["pipeline"].(map[string]any); ok {
			if callbacks.onPipelineUpdate != nil {
				callbacks.onPipelineUpdate(pipelineData, eventRunId)
			}
		}
	})

	if flags.VerboseFlag && callbacks.onVerbose != nil {
		callbacks.onVerbose("Starting run log stream")
	}

	ioService.Emit("Stream:RunLog:Start", map[string]any{
		"runId": runID,
	})

	<-done

	ioService.Emit("Stream:RunLog:Stop", map[string]any{
		"runId": runID,
	})

	return nil
}

func runWithTUI(orgId, deploymentId string, run *models.DeploymentRun, appUrl string, service *Deployment.DeploymentService) {
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

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := streamDeployEvents(orgId, run.ID, deployCallbacks{
			onVerbose: func(msg string) {
				statusDisplay.Send(Deployment.VerboseMessage{Content: msg})
			},
			onStatus: func(runId, status string) {
				statusDisplay.Send(Deployment.RunMessageData{
					Type:   "run",
					RunId:  runId,
					Id:     runId,
					Status: status,
				})
			},
			onPipelineUpdate: func(pipelineData map[string]any, runId string) {
				extractStatusUpdates(pipelineData, runId, statusDisplay)
			},
			onComplete: func(status string) {
				statusDisplay.Quit()
			},
			onError: func(err error) {
				statusDisplay.Send(Deployment.ErrorMessage{Error: err})
			},
		})

		if err != nil {
			statusDisplay.Send(Deployment.ErrorMessage{Error: err})
		}
	}()

	statusDisplay.Run()
	wg.Wait()
}

func runWithoutTUI(orgId string, deploymentId string, run *models.DeploymentRun, appUrl string, service *Deployment.DeploymentService) {
	printer.Print(fmt.Sprintf("Deployment URL: %s", appUrl))
	printer.Print("Monitoring deployment progress...")

	err := streamDeployEvents(orgId, run.ID, deployCallbacks{
		onVerbose: func(msg string) {
			printer.Print(fmt.Sprintf("  [verbose] %s", msg))
		},
		onStatus: func(runId, status string) {
			printer.Print(fmt.Sprintf("Deployment: %s", status))
		},
		onPipelineUpdate: func(pipelineData map[string]any, runId string) {
			printPipelineUpdates(pipelineData)
		},
		onComplete: func(status string) {
			switch status {
			case "succeeded":
				printer.Success("Deployment completed successfully")
			case "failed":
				printer.Print("Deployment failed")
			case "canceled":
				printer.Warning("Deployment was canceled")
			}
		},
		onError: nil,
	})

	if err != nil {
		printer.Print(fmt.Sprintf("Error: %v", err))
	}
}

func printPipelineUpdates(pipelineData map[string]any) {
	if steps, ok := pipelineData["steps"].([]any); ok {
		for _, stepRaw := range steps {
			step, ok := stepRaw.(map[string]any)
			if !ok {
				continue
			}

			stepName, _ := step["name"].(string)
			stepStatus, _ := step["status"].(string)

			if stepName != "" && stepStatus != "" {
				printer.Print(fmt.Sprintf("  Step %s: %s", stepName, stepStatus))
			}

			if tasks, ok := step["tasks"].([]any); ok {
				for _, taskRaw := range tasks {
					task, ok := taskRaw.(map[string]any)
					if !ok {
						continue
					}

					taskType, _ := task["type"].(string)
					taskStatus, _ := task["status"].(string)

					if taskType != "" && taskStatus != "" {
						printer.Print(fmt.Sprintf("    Task %s: %s", taskType, taskStatus))
					}
				}
			}
		}
	}
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

func init() {
	DeployCmd.Flags().BoolVar(&noBuild, "no-build", false, "Skip the build step")
	DeployCmd.Flags().StringVarP(&flags.DockerfileFlag, "dockerfile", "f", "", "Path to Dockerfile (e.g., ./Dockerfile or ./docker/Dockerfile.prod)")
}
