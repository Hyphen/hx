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
}
