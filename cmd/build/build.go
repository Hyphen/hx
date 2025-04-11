package build

import (
	"fmt"

	"github.com/Hyphen/cli/internal/build"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/dockerutil"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	printer *cprint.CPrinter
)

var BuildCmd = &cobra.Command{
	Use:   "build ",
	Short: "Run a build and post it to hyphen",
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
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		service := build.NewService()

		// TODO: this probably should error if there is no hxkey
		// file it just means they aren't using env or are using the new
		// cert store service.
		// grab the manifest to get app details
		manifest, err := manifest.Restore()
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		if manifest.IsMonorepoProject() {
			printer.Error(cmd, fmt.Errorf("monorepo projects are not supported yet"))
			return
		}

		// Check for docker
		isDockerAvailable := dockerutil.IsDockerAvaliable()
		if !isDockerAvailable {
			printer.Error(cmd, fmt.Errorf("docker is not installed or not in PATH"))
			return
		}
		printer.Info("Docker is available. Proceeding with the build process.")

		// Try to find a docker file to run
		dockerfilePath, err := dockerutil.FindDockerFile()
		if err != nil || dockerfilePath == "" {
			printer.Error(cmd, fmt.Errorf("no docker file found. Dynamic builds are not supported yet"))
			return
		}

		registerUrl := "deploydevelopmentregistry.azurecr.io"
		commitSha := "1b3d5f"
		//imageUrl := "us-docker.pkg.dev/hyphenai/public/deploy-demo"

		// Run build on the docker file

		name, dockerBuildOutput, err := dockerutil.Build(dockerfilePath, *manifest.AppAlternateId, commitSha)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to build docker image: %w", err))
			return
		}

		printer.Info(dockerBuildOutput)
		// push the image to a register

		containerUrl, err := dockerutil.Push(name, registerUrl)

		// Tell Hyphen about the build
		service.CreateBuild(manifest.OrganizationId, *manifest.AppId, commitSha, containerUrl)
	},
}
