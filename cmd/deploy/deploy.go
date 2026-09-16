package deploy

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Hyphen/cli/internal/build"
	"github.com/Hyphen/cli/internal/config"
	Deployment "github.com/Hyphen/cli/internal/deployment"
	"github.com/Hyphen/cli/internal/env"
	hyphenapp "github.com/Hyphen/cli/internal/hyphenApp"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/socketio"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	noBuild          bool
	envFlag          string
	projectFlag      string
	appsFlag         string
	sitesFlag        string
	buildType        string
	outputFormatFlag string
	printer          *cprint.CPrinter
)

var DeployCmd = &cobra.Command{
	Use:   "deploy [deploymentId] [flags] [site-directory]",
	Short: "Run a deployment",
	Long: `
Run a deployment by ID, or omit it to deploy the development environment.

If no deploymentId is provided, the CLI will use the current project config to
find the development environment deployment. If it doesn't exist yet, it will
be created automatically.

Usage:
  hyphen deploy [deploymentId] [flags] [site-directory]

Examples:
  hyphen deploy                  # deploys the dev environment (auto-detected)
  hyphen deploy depl_abc123      # deploys a specific deployment by ID
  hyphen deploy --type static ./dist
  hyphen deploy depl_abc123 --no-build --apps api:latest --sites website:abld_123

--type selects the local build (docker by default). Static builds require the
website directory as the final argument, unless --no-build is used. --apps and
--sites select containers and static sites separately and may be combined.

Use 'hyphen deploy --help' for more information about available flags.
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if _, _, err := deployArguments(args, buildType, noBuild, flags.DockerfileFlag); err != nil {
			return err
		}
		_, err := parseSelections(appsFlag, sitesFlag)
		return err
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		printer.SetFormat(outputFormatFlag)

		result, runErr := runDeployBody(cmd, args)

		if runErr != nil {
			if printer.IsJSON() {
				if result == nil {
					result = map[string]any{}
				}
				// Preserve a real terminal status (e.g. "canceled") if
				// runDeployBody set one before failing; otherwise mark
				// the run as failed so JSON consumers see a consistent
				// non-empty status.
				if _, ok := result["status"]; !ok {
					result["status"] = "failed"
				}
				result["reason"] = runErr.Error()
				if emitErr := printer.Emit(result); emitErr != nil {
					fmt.Fprintf(os.Stderr, "failed to emit JSON output: %v\n", emitErr)
				}
				// Silence cobra's error/usage output so it doesn't
				// pollute stdout after we've emitted the JSON
				// payload — the error is already surfaced in the
				// payload's "reason" field.
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
			} else {
				// Human mode already streamed the failure context to
				// stdout, so silence cobra's redundant "Error: ..."
				// line — but still return the error to drive a
				// non-zero exit code.
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
			}
			return runErr
		}

		return printer.Emit(result)
	},
}

// runDeployBody performs the deployment and returns a result map suitable
// for emitting as the --output=json payload. The result is accumulated as
// the command progresses so that an early error still returns everything
// known so far (deploymentId, runId, deploymentUrl as available); the outer
// RunE wrapper tags it with status=failed and reason on error.
func runDeployBody(cmd *cobra.Command, args []string) (map[string]any, error) {
	result := map[string]any{}
	deploymentID, directory, err := deployArguments(args, buildType, noBuild, flags.DockerfileFlag)
	if err != nil {
		return result, err
	}
	selections, err := parseSelections(appsFlag, sitesFlag)
	if err != nil {
		return result, err
	}

	orgId, err := flags.GetOrganizationID()
	if err != nil {
		return result, err
	}

	service := Deployment.NewService()

	var selectedDeployment models.Deployment

	if deploymentID == "" {
		cfg, err := config.RestoreLocalConfig()
		if err != nil {
			return result, fmt.Errorf("failed to restore config: %w", err)
		}

		projectId := projectFlag
		if cfg.ProjectId != nil {
			projectId = *cfg.ProjectId
		}
		if projectId == "" {
			return result, fmt.Errorf("project id not found in config")
		}

		var envId string
		if envFlag != "" {
			envId = envFlag
		} else {
			envService := env.NewService()
			devEnv, devEnvErr := envService.GetDevelopmentEnvironment(orgId, projectId)
			if errors.Is(devEnvErr, errors.ErrNotFound) {
				return result, fmt.Errorf("no development environment found for this project")
			}
			if devEnvErr != nil {
				return result, fmt.Errorf("failed to get development environment: %w", devEnvErr)
			}
			envId = devEnv.ID
		}

		projectService := projects.NewService(orgId)
		deployment, deploymentErr := projectService.GetEnvironmentDeployment(projectId, envId)
		if deploymentErr != nil && !errors.Is(deploymentErr, errors.ErrNotFound) {
			return result, fmt.Errorf("failed to get deployment for environment: %w", deploymentErr)
		}

		if errors.Is(deploymentErr, errors.ErrNotFound) || deployment.ID == "" {
			project, err := projectService.GetProject(projectId)
			if err != nil {
				return result, fmt.Errorf("failed to get project: %w", err)
			}
			if cfg.AppId == nil {
				return result, fmt.Errorf("app id not found in config")
			}

			name := deploymentNamePart(project.AlternateID, 25)

			siteIntegrationID := ""
			initialKind := buildType
			for _, selected := range selections {
				if matchesHxApp(selected.ID, cfg) {
					initialKind = selected.Kind
				}
			}
			if initialKind == "static" {
				registry, err := build.NewService().FindSiteRegistry(orgId, projectId)
				if err != nil {
					return result, err
				}
				siteIntegrationID = registry.OrganizationIntegration.Id
			}
			newDeployment, err := service.CreateEnvironmentDeployment(orgId, projectId, envId, *cfg.AppId, name, name, "", siteIntegrationID)
			if err != nil {
				return result, fmt.Errorf("failed to create deployment: %w", err)
			}
			selectedDeployment = *newDeployment
		} else {
			selectedDeployment = deployment
		}
	} else {
		deployment, err := service.GetDeployment(orgId, deploymentID)
		if err != nil {
			return result, fmt.Errorf("failed to get deployment: %w", err)
		}

		selectedDeployment = *deployment
	}

	result["deploymentId"] = selectedDeployment.ID

	cfg, cfgErr := config.RestoreLocalConfig()
	if cfgErr != nil && !os.IsNotExist(cfgErr) {
		return result, fmt.Errorf("failed to restore config: %w", cfgErr)
	}
	explicitSelection := len(selections) > 0
	if !explicitSelection && cfg.AppId != nil {
		kind := buildType
		if noBuild {
			if _, existingKind := deploymentMember(*cfg.AppId, selectedDeployment); existingKind != "" {
				kind = existingKind
			}
		}
		selections = []selection{{ID: *cfg.AppId, Kind: kind}}
	}
	if buildType == "static" && !noBuild && !selectsLocalBuild(selections, cfg, buildType) {
		return result, fmt.Errorf("static build requires the configured .hx app to be selected without a build selector; use --no-build to deploy existing builds")
	}
	if !noBuild && selectsLocalBuild(selections, cfg, buildType) && (cfg.OrganizationId != orgId || cfg.ProjectId == nil || *cfg.ProjectId != selectedDeployment.Project.ID) {
		return result, fmt.Errorf("local .hx app must belong to the deployment organization and project")
	}
	selectedDeployment, err = ensureSelections(service, orgId, selectedDeployment, selections)
	if err != nil {
		return result, err
	}
	if !explicitSelection {
		selections = allSelections(selectedDeployment)
	}

	if !selectedDeployment.IsReady {
		printer.Print("❌ There are issues blocking this deployment from being run.")
		issues := []string{}
		for _, issue := range selectedDeployment.ReadinessIssues {
			var line string
			if issue.Cloud != "" {
				line = fmt.Sprintf("%s (%s)", issue.Error, issue.Cloud)
			} else {
				line = issue.Error
			}
			printer.Print("  • " + line)
			issues = append(issues, line)
		}
		return result, fmt.Errorf("deployment not ready: %s", strings.Join(issues, "; "))
	}

	// Match preview if preview flag is provided
	var previewId, previewHash string
	if flags.PreviewNameFlag != "" {
		matchedPreviews := []models.DeploymentPreview{}
		for _, p := range selectedDeployment.Previews {
			// If prefix is provided, also match by hostPrefix
			if p.Name == flags.PreviewNameFlag {
				if flags.PreviewPrefixFlag != "" {
					if p.HostPrefix == flags.PreviewPrefixFlag {
						matchedPreviews = append(matchedPreviews, p)
					}
				} else {
					matchedPreviews = append(matchedPreviews, p)
				}
			}
		}

		if len(matchedPreviews) == 0 {
			if flags.PreviewPrefixFlag == "" {
				return result, fmt.Errorf("no preview found with name '%s', please specify --prefix flag to create a new preview", flags.PreviewNameFlag)
			}
			newPreview, err := service.CreatePreview(orgId, selectedDeployment, flags.PreviewNameFlag, flags.PreviewPrefixFlag)
			if err != nil {
				return result, fmt.Errorf("failed to create preview: %w", err)
			}
			previewId = newPreview.ID
			previewHash = newPreview.HostPrefix
		} else if len(matchedPreviews) > 1 {
			return result, fmt.Errorf("multiple previews found with name '%s', please specify --prefix flag to disambiguate", flags.PreviewNameFlag)
		} else {
			previewId = matchedPreviews[0].ID
			previewHash = matchedPreviews[0].HostPrefix
		}
	}

	appSources, err := deploymentSources(selectedDeployment, selections)
	if err != nil {
		return result, err
	}
	if !noBuild && (!explicitSelection || selectsLocalBuild(selections, cfg, buildType)) {
		buildResult, err := build.NewService().RunBuild(cmd, printer, build.Options{
			Type: buildType, Directory: directory, EnvironmentID: selectedDeployment.ProjectEnvironment.ID,
			Preview: flags.PreviewNameFlag, PreviewHash: previewHash,
			Verbose: flags.VerboseFlag, DockerfilePath: flags.DockerfileFlag,
		})
		if err != nil {
			return result, err
		}
		for i := range appSources {
			if appSources[i].AppId == buildResult.App.ID {
				appSources[i].Build = ""
				appSources[i].BuildId = buildResult.Id
			}
		}
	}

	printer.Print(fmt.Sprintf("Running %s", selectedDeployment.Name))

	run, err := service.CreateRun(orgId, selectedDeployment.ID, appSources, previewId)
	if err != nil {
		return result, fmt.Errorf("failed to create run: %w", err)
	}
	if run == nil {
		return result, fmt.Errorf("failed to create run: empty response")
	}

	appUrl := hyphenapp.DeploymentRunLinkForRun(orgId, &selectedDeployment, run)

	result["runId"] = run.ID
	result["deploymentUrl"] = appUrl

	var finalStatus string
	var streamErr error
	if shouldUseTUI() {
		finalStatus, streamErr = runWithTUI(orgId, selectedDeployment.ID, run, appUrl, service)
	} else {
		finalStatus, streamErr = runWithoutTUI(orgId, selectedDeployment.ID, run, appUrl, service)
	}

	if finalStatus != "" {
		result["status"] = finalStatus
	}

	if streamErr != nil {
		return result, fmt.Errorf("failed to monitor deployment: %w", streamErr)
	}

	if finalStatus != "succeeded" {
		return result, fmt.Errorf("deployment ended with status %q", finalStatus)
	}

	return result, nil
}

func shouldUseTUI() bool {
	if outputFormatFlag == cprint.FormatJSON {
		return false
	}
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

func runWithTUI(orgId, deploymentId string, run *models.DeploymentRun, appUrl string, service *Deployment.DeploymentService) (string, error) {
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

	var finalStatus string
	var streamErr error
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
				finalStatus = status
				statusDisplay.Quit()
			},
			onError: func(err error) {
				statusDisplay.Send(Deployment.ErrorMessage{Error: err})
			},
		})

		if err != nil {
			streamErr = err
			statusDisplay.Send(Deployment.ErrorMessage{Error: err})
			statusDisplay.Quit()
		}
	}()

	statusDisplay.Run()
	wg.Wait()
	return finalStatus, streamErr
}

func runWithoutTUI(orgId string, deploymentId string, run *models.DeploymentRun, appUrl string, service *Deployment.DeploymentService) (string, error) {
	printer.Print(fmt.Sprintf("Deployment URL: %s", appUrl))
	printer.Print("Monitoring deployment progress...")

	var finalStatus string
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
			finalStatus = status
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
		return finalStatus, err
	}
	return finalStatus, nil
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

func deploymentNamePart(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

func matchesHxApp(identifier string, cfg config.Config) bool {
	if cfg.AppId != nil && *cfg.AppId == identifier {
		return true
	}
	if cfg.AppAlternateId != nil && *cfg.AppAlternateId == identifier {
		return true
	}
	return false
}

func init() {
	DeployCmd.Flags().BoolVar(&noBuild, "no-build", false, "Skip local building; use selected builds (latest when unspecified)")
	DeployCmd.Flags().StringVar(&buildType, "type", "docker", "Local build type: docker or static (requires a final site directory argument unless --no-build)")
	DeployCmd.Flags().StringVar(&sitesFlag, "sites", "", "Comma-separated static sites to deploy, with the same build selectors as --apps")
	DeployCmd.Flags().StringVarP(&flags.DockerfileFlag, "dockerfile", "f", "", "Path to Dockerfile (e.g., ./Dockerfile or ./docker/Dockerfile.prod)")
	DeployCmd.Flags().StringVarP(&flags.PreviewNameFlag, "preview", "r", "", "Preview name to deploy to")
	DeployCmd.Flags().StringVarP(&flags.PreviewPrefixFlag, "prefix", "x", "", "Host prefix for the preview deployment")
	DeployCmd.Flags().StringVar(&envFlag, "env", "", "Environment to deploy (defaults to the environment flagged as the \"development\" type)")
	DeployCmd.Flags().StringVar(&projectFlag, "project", "", "Project to deploy (defaults to project ID in hx config)")
	DeployCmd.Flags().StringVar(&appsFlag, "apps", "", "Comma-separated list of apps to deploy, each optionally specifying a build (e.g. app1,app2:abld_xxxx,app3:latest,app4:lastDeployed,app5:latestPreview)")
	DeployCmd.Flags().StringVar(&outputFormatFlag, "output", "", "Output format. Set to \"json\" to emit a JSON object with deploymentId, runId, deploymentUrl, status, and a messages array on completion instead of streaming human-readable progress.")
}
