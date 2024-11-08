package rotatekey

import (
	"fmt"

	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/cprint"
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
	// Display warning and prompt for confirmation
	printer.Warning("You are about to rotate the encryption key. This action is irreversible and will affect all environments.")
	printer.Info("Make sure you have a backup of your current configuration before proceeding.")
	printer.Info("This action will:")
	printer.Info("  1. Generate a new encryption key")
	printer.Info("  2. Re-encrypt all environment variables with the new key")
	printer.Info("  3. Update the local configuration with the new key")
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

	currentManifest, err := manifest.Restore()
	if err != nil {
		return fmt.Errorf("failed to restore manifest: %w", err)
	}

	newManifestSecret, err := getNewManifestSecret()
	if err != nil {
		return err
	}

	manifest.UpsertLocalSecret(newManifestSecret)

	push.Silent = true
	push.RunPush([]string{}, currentManifest.SecretKeyId)

	printer.Success("Key rotation completed successfully.")
	printer.Info("New encryption key has been generated and all environments have been updated.")
	printer.Info("Please ensure all team members pull the latest changes.")

	return nil
}

func getNewManifestSecret() (manifest.Secret, error) {
	//generate new key
	newSecretKey, err := secretkey.New()
	if err != nil {
		return manifest.Secret{}, err
	}

	return manifest.NewSecret(newSecretKey), nil
}
