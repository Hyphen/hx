package code

import (
	"github.com/Hyphen/cli/cmd/code/docker"
	"github.com/Hyphen/cli/internal/user"
	"github.com/spf13/cobra"
)

var CodeCmd = &cobra.Command{
	Use:   "code",
	Short: "Work with the Hyphen Code",
	Long:  `Use the Hyphen Code to perform tasks both locally and remotely.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
}

func init() {

	CodeCmd.AddCommand(docker.DockerCmd)
}
