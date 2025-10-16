package docker

import (
	"github.com/Hyphen/cli/internal/code"
	"github.com/Hyphen/cli/pkg/cprint"
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
		coder := code.NewService()
		err := coder.GenerateDocker(printer, cmd)
		if err != nil {
			printer.Error(cmd, err)
			return
		}
	},
}
