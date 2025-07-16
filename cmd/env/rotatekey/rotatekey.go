package rotatekey

import (
	"fmt"

	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/internal/secret"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var (
	forceFlag bool
	printer   *cprint.CPrinter
)

var RotateCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Rotate the encryption key and update all environments",
	Long:  `This command rotates the encryption key and updates all environments with the new key.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		if err := runRotateKey(cmd); err != nil {
			printer.Error(cmd, err)
		}
	},
}

func init() {
	RotateCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite of locally modified environment files")
}

func runRotateKey(cmd *cobra.Command) error {
	// Get organization and project info for detection
	organizationId, err := flags.GetOrganizationID()
	if err != nil {
		return errors.Wrap(err, "Failed to get organization ID")
	}

	projectId, err := flags.GetProjectID()
	if err != nil {
		return errors.Wrap(err, "Failed to get project ID")
	}

	// Get the current location
	_, location, err := secret.LoadSecret(organizationId, projectId)
	if err != nil {
		return errors.Wrap(err, "unable to load secret for rotation")
	}

	if location == secret.SecretLocationNone {
		return fmt.Errorf("there is currently no secret to rotate for this application")
	}

	locationDesc := "remotely"
	if location == secret.SecretLocationLocal {
		locationDesc = "locally"
	}

	// Display warning and prompt for confirmation
	printer.Warning("You are about to rotate the encryption key. This action is irreversible and will affect all environments.")
	printer.Info("Make sure you have a backup of your current configuration before proceeding.")
	printer.Info("This action will:")
	printer.Info("  1. Generate a new encryption key")
	printer.Info("  2. Re-encrypt all environment variables with the new key")
	printer.Info(fmt.Sprintf("  3. Update the secret key stored %s", locationDesc))
	printer.Info("\nAre you absolutely sure you want to proceed?")

	response := prompt.PromptYesNo(cmd, "Rotate encryption key?", false)
	if !response.Confirmed {
		printer.Info("Key rotation cancelled.")
		return nil
	}

	// Proceed with key rotation
	printer.Info("Proceeding with key rotation...")

	//Get all the envs
	pull.Silent = true
	if err := pull.RunPull([]string{}, forceFlag); err != nil {
		return err
	}

	err = secret.RotateSecret()
	if err != nil {
		return err
	}

	push.Silent = true
	push.RunPush([]string{})

	printer.Success("Key rotation completed successfully.")
	printer.Info(fmt.Sprintf("New encryption key has been generated and stored %s", locationDesc))
	printer.Info("All environments have been updated with the new key.")
	printer.Info("Please ensure all team members pull the latest changes.")

	return nil
}
