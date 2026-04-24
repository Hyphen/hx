package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCmd(t *testing.T) {
	t.Run("has_an_env_flag", func(t *testing.T) {
		flag := BuildCmd.Flags().Lookup("env")

		assert.NotNil(t, flag, "build command should have an --env flag to specify the environment")
	})

	t.Run("has_an_output_flag", func(t *testing.T) {
		flag := BuildCmd.Flags().Lookup("output")

		assert.NotNil(t, flag, "build command should have an --output flag")
	})

	t.Run("defaults_the_output_flag_to_empty_string", func(t *testing.T) {
		flag := BuildCmd.Flags().Lookup("output")

		assert.Equal(t, "", flag.DefValue)
	})
}
