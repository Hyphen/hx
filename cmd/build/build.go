package build

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/build"
	hyphenapp "github.com/Hyphen/cli/internal/hyphenApp"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	printer          *cprint.CPrinter
	outputFormatFlag string
)

var BuildCmd = &cobra.Command{
	Use:   "build ",
	Short: "Run a build and post it to hyphen",
	Long: `
The build command runs a build and uploads it to hyphen without deploying.

Usage:
	hyphen build [flags]

Examples:
hyphen build

Use 'hyphen build --help' for more information about available flags.
`,
	Args: cobra.RangeArgs(0, 1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		printer.SetFormat(outputFormatFlag)

		result, runErr := runBuild(cmd)
		if runErr != nil {
			if printer.IsJSON() {
				emitFailure(printer, map[string]any{}, runErr)
				// Silence cobra's error/usage output so it doesn't
				// pollute stdout after we've emitted the JSON
				// payload — the error is already surfaced in the
				// payload's "reason" field.
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
			}
			return runErr
		}

		return printer.Emit(result)
	},
}

// emitFailure writes the failure JSON payload to stdout and falls back to
// stderr if Emit itself fails. Ensures we never silently drop the error in
// JSON mode.
func emitFailure(p *cprint.CPrinter, partial map[string]any, err error) {
	partial["status"] = "failed"
	partial["reason"] = err.Error()
	if emitErr := p.Emit(partial); emitErr != nil {
		fmt.Fprintf(os.Stderr, "failed to emit JSON output: %v\n", emitErr)
	}
}

// runBuild performs the build and returns the result fields to emit, or an
// error. Keeping the body here (rather than inline in RunE) lets the RunE
// wrapper uniformly translate any error into a JSON failure payload when
// --output=json is set.
func runBuild(cmd *cobra.Command) (map[string]any, error) {
	service := build.NewService()
	result, err := service.RunBuild(cmd, printer, flags.EnvironmentFlag, flags.VerboseFlag, flags.DockerfileFlag, flags.PreviewNameFlag)
	if err != nil {
		return nil, err
	}

	url := hyphenapp.ApplicationBuildLink(result.Organization.ID, result.Project.ID, result.App.ID, result.Id)
	printer.Info("Build successful: " + url)

	return map[string]any{
		"status":         "succeeded",
		"buildId":        result.Id,
		"appId":          result.App.ID,
		"projectId":      result.Project.ID,
		"organizationId": result.Organization.ID,
		"buildUrl":       url,
	}, nil
}

func init() {
	BuildCmd.Flags().StringVarP(&flags.DockerfileFlag, "dockerfile", "f", "", "Path to Dockerfile (e.g., ./Dockerfile or ./docker/Dockerfile.prod)")
	BuildCmd.Flags().StringVarP(&flags.EnvironmentFlag, "env", "e", "", "Environment ID for the build")
	BuildCmd.Flags().StringVarP(&flags.PreviewNameFlag, "preview", "r", "", "Preview name to associate with this build")
	BuildCmd.Flags().StringVar(&outputFormatFlag, "output", "", "Output format. Set to \"json\" to emit a JSON object with status, buildId, appId, projectId, organizationId, buildUrl, reason (on failure), and a messages array.")
}
