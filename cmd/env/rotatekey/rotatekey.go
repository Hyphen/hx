package rotatekey

import (
	"fmt"

	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var (
	forceFlag bool
)

var RotateCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Rotate the encryption key and update all environments",
	Long:  `This command rotates the encryption key and updates all environments with the new key.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRotateKey(cmd); err != nil {
			cprint.Error(cmd, err)
		}
	},
}

func init() {
	RotateCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite of locally modified environment files")
}

func runRotateKey(cmd *cobra.Command) error {
	// Display warning and prompt for confirmation
	cprint.Warning("You are about to rotate the encryption key. This action is irreversible and will affect all environments.")
	cprint.Warning("Make sure you have a backup of your current configuration before proceeding.")
	cprint.Warning("This action will:")
	cprint.Warning("  1. Generate a new encryption key")
	cprint.Warning("  2. Re-encrypt all environment variables with the new key")
	cprint.Warning("  3. Update the local configuration with the new key")
	cprint.Warning("\nAre you absolutely sure you want to proceed?")

	response := prompt.PromptYesNo(cmd, "Rotate encryption key?", false)
	if !response.Confirmed {
		cprint.Info("Key rotation cancelled.")
		return nil
	}

	// Proceed with key rotation
	cprint.Info("Proceeding with key rotation...")

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

	manifest.UpsertLocalManifestSecret(newManifestSecret)

	push.Silent = true
	push.RunPush([]string{}, currentManifest.SecretKeyId)

	cprint.Success("Key rotation completed successfully.")
	cprint.Info("New encryption key has been generated and all environments have been updated.")
	cprint.Info("Please ensure all team members pull the latest changes.")

	return nil
}

func getNewManifestSecret() (manifest.ManifestSecret, error) {
	//generate new key
	newSecretKey, err := secretkey.New()
	if err != nil {
		return manifest.ManifestSecret{}, err
	}

	return manifest.NewSecret(newSecretKey), nil
}
