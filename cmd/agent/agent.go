package agent

import (
	"github.com/Hyphen/cli/cmd/agent/docker"
	"github.com/Hyphen/cli/internal/user"
	"github.com/spf13/cobra"
)

var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Work with the Hyphen Agent",
	Long:  `Use the Hyphen Agent to perform tasks but locally and remotely.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
}

func init() {

	AgentCmd.AddCommand(docker.DockerCmd)
}
