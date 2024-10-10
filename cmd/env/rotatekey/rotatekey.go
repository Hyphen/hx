package rotatekey

import (
	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/cprint"
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
		if err := runRotateKey(); err != nil {
			cprint.Error(cmd, err)
		}
	},
}

func init() {
	RotateCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite of locally modified environment files")
}

func runRotateKey() error {
	//Get all the envs
	if err := pull.RunPull([]string{}, forceFlag); err != nil {
		return err
	}

	newManifestSecret, err := getNewManifestSecret()
	if err != nil {
		return err
	}

	m, err := manifest.Restore()

	//KeyRotatePutEnv
	if err := putWithRotationKey(m, newManifestSecret); err != nil {
		return err
	}

	if err := saveNewSecret(m, newManifestSecret); err != nil {
		return err
	}

	return nil
}

func putWithRotationKey(m manifest.Manifest, newSecret manifest.ManifestSecret) error {
	//KeyRotatePutEnv
	return nil
}

func saveNewSecret(m manifest.Manifest, newSecret manifest.ManifestSecret) error {
	//KeyRotateSaveSecret
	return nil
}

func getNewManifestSecret() (manifest.ManifestSecret, error) {
	//generate ney key
	newSecretKey, err := secretkey.New()
	if err != nil {
		return manifest.ManifestSecret{}, err
	}

	return manifest.NewSecret(newSecretKey), nil
}
