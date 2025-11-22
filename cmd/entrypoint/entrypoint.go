package entrypoint

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

//go:embed hyphen-entrypoint.sh
var hyphenEntrypoint string

var (
	forceFlag bool
	printer   *cprint.CPrinter
)

var EntrypointCmd = &cobra.Command{
	Use:   "entrypoint",
	Short: "Copy hyphen-entrypoint.sh to the current folder",
	Long: `
hyphen-entrypoint.sh is a Linux shell script intended to be used in Docker container
environments. On startup, it will set up your container environment variables based
on the deployment environment you have selected.

This command will place a copy of the current hyphen-entrypoint.sh file into the
current folder.
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		if err := CreateEntrypoint(forceFlag); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	EntrypointCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite of hyphen-entrypoint.sh if it already exists")
}

func CreateEntrypoint(force bool) error {
	if !force {
		if _, err := os.Stat("hyphen-entrypoint.sh"); err == nil {
			return errors.New("hyphen-entrypoint.sh already exists; use --force to overwrite")
		}
	}

	file, err := os.OpenFile("hyphen-entrypoint.sh", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("error creating hyphen-entrypoint.sh: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(hyphenEntrypoint)
	if err != nil {
		return fmt.Errorf("error writing to hyphen-entrypoint.sh: %w", err)
	}

	return nil
}
