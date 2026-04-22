package deploy

import (
	"testing"

	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/stretchr/testify/assert"
)

func TestDeployCmd(t *testing.T) {
	t.Run("has_an_output_flag", func(t *testing.T) {
		flag := DeployCmd.Flags().Lookup("output")

		assert.NotNil(t, flag, "deploy command should have an --output flag")
	})

	t.Run("defaults_the_output_flag_to_empty_string", func(t *testing.T) {
		flag := DeployCmd.Flags().Lookup("output")

		assert.Equal(t, "", flag.DefValue)
	})
}

func TestShouldUseTUI(t *testing.T) {
	t.Run("returns_false_when_output_format_is_json", func(t *testing.T) {
		originalFlag := outputFormatFlag
		outputFormatFlag = cprint.FormatJSON
		t.Cleanup(func() { outputFormatFlag = originalFlag })

		result := shouldUseTUI()

		assert.False(t, result)
	})
}
