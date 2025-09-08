package docker

import (
	"fmt"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
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
		if err := RunDockerGen(args); err != nil {
			printer.Error(cmd, err)
		}
	},
}

func RunDockerGen(args []string) error {
	config, err := config.RestoreConfig()
	if err != nil {
		return err
	}

	if config.IsMonorepoProject() {
		return fmt.Errorf("docker generation for monorepos is not supported yet")
	}

	printer.Print("Generating Dockerfile...")
	return nil
}
