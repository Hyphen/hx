package update

import (
	"os"
	"testing"

	internalconfig "github.com/Hyphen/cli/internal/config"
)

func TestShouldSkipAutoUpdateCommandPath(t *testing.T) {
	testCases := []struct {
		name        string
		commandPath string
		expected    bool
	}{
		{name: "root", commandPath: "hyphen", expected: true},
		{name: "update command", commandPath: "hyphen update", expected: true},
		{name: "version command", commandPath: "hyphen version", expected: true},
		{name: "config command", commandPath: "hyphen config auto-update", expected: true},
		{name: "regular command", commandPath: "hyphen env pull", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := shouldSkipAutoUpdateCommandPath(tc.commandPath)
			if actual != tc.expected {
				t.Fatalf("expected %v for command path %q, got %v", tc.expected, tc.commandPath, actual)
			}
		})
	}
}

func TestIsTruthyEnvValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "empty", input: "", expected: false},
		{name: "false", input: "false", expected: false},
		{name: "zero", input: "0", expected: false},
		{name: "true", input: "true", expected: true},
		{name: "github-actions", input: "true", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := isTruthyEnvValue(tc.input)
			if actual != tc.expected {
				t.Fatalf("expected %v for input %q, got %v", tc.expected, tc.input, actual)
			}
		})
	}
}

func TestIsCIEnvironment(t *testing.T) {
	ciEnvVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"BUILDKITE",
		"CIRCLECI",
		"GITLAB_CI",
		"JENKINS_URL",
		"TF_BUILD",
		"TEAMCITY_VERSION",
		"CODEBUILD_BUILD_ID",
	}

	for _, key := range ciEnvVars {
		t.Setenv(key, "")
	}

	t.Run("detects github actions", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		if !isCIEnvironment() {
			t.Fatalf("expected CI environment to be detected when GITHUB_ACTIONS is set")
		}
	})

	t.Run("treats CI=false as non-ci", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("CI", "false")
		if isCIEnvironment() {
			t.Fatalf("expected CI environment to be false when CI is explicitly false")
		}
	})
}

func TestIsAutoUpdateEnabledFromConfig(t *testing.T) {
	originalRestore := restoreGlobalConfig
	t.Cleanup(func() {
		restoreGlobalConfig = originalRestore
	})

	t.Run("defaults to enabled when config is missing", func(t *testing.T) {
		restoreGlobalConfig = func() (internalconfig.Config, error) {
			return internalconfig.Config{}, os.ErrNotExist
		}

		enabled, err := isAutoUpdateEnabledFromConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Fatalf("expected auto-update to be enabled when config is missing")
		}
	})

	t.Run("returns disabled when config says disabled", func(t *testing.T) {
		disabled := true
		restoreGlobalConfig = func() (internalconfig.Config, error) {
			return internalconfig.Config{
				AutoUpdateDisabled: &disabled,
			}, nil
		}

		enabled, err := isAutoUpdateEnabledFromConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Fatalf("expected auto-update to be disabled")
		}
	})
}
