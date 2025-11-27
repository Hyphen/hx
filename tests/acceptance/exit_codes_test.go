package acceptance

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIExitCodes(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	t.Run("version_command_exits_with_zero_on_success", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "version")
		cmd.Dir = projectRoot
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		assert.NoError(t, err, "version command should succeed")
		assert.Equal(t, 0, cmd.ProcessState.ExitCode(), "exit code should be 0")
		assert.Empty(t, stderr.String(), "stderr should be empty on success")
		assert.NotContains(t, stdout.String(), "ERROR:", "stdout should not contain error messages")
	})

	t.Run("invalid_command_exits_with_code_1_and_prints_error", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "nonexistent-command")
		cmd.Dir = projectRoot
		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined

		err := cmd.Run()

		assert.Error(t, err, "invalid command should exit with non-zero code")
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 1, exitErr.ExitCode(), "exit code should be 1")
		}
		output := combined.String()
		assert.Contains(t, output, "unknown command", "error message should be printed, got: %s", output)
	})

	t.Run("command_requiring_auth_exits_with_code_1_when_not_authenticated", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "app", "list")
		cmd.Dir = projectRoot
		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined

		err := cmd.Run()

		assert.Error(t, err, "unauthenticated command should exit with non-zero code")
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 1, exitErr.ExitCode(), "exit code should be 1")
		}
		output := combined.String()
		assert.Contains(t, output, "ERROR:", "error should be formatted with ERROR prefix")
		assert.Contains(t, output, "authenticate", "error message should indicate authentication is required")
	})

	t.Run("command_requiring_auth_exits_with_code_1_when_not_authenticated_verbose", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "app", "list", "--verbose")
		cmd.Dir = projectRoot
		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined

		err := cmd.Run()

		assert.Error(t, err, "unauthenticated command should exit with non-zero code")
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 1, exitErr.ExitCode(), "exit code should be 1")
		}
		output := combined.String()
		assert.Contains(t, output, "error -", "error should be formatted with verbose error prefix")
		assert.Contains(t, output, "authenticate", "error message should indicate authentication is required")
	})

	t.Run("command_with_missing_required_arg_exits_with_code_1", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "app", "create")
		cmd.Dir = projectRoot
		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined

		err := cmd.Run()

		assert.Error(t, err, "command with missing required arg should exit with non-zero code")
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 1, exitErr.ExitCode(), "exit code should be 1")
		}
		output := combined.String()
		assert.Contains(t, output, "ERROR:", "error should be formatted with ERROR prefix")
		assert.True(t,
			strings.Contains(output, "accepts") || strings.Contains(output, "arg"),
			"error message should indicate argument issue")
	})
}
