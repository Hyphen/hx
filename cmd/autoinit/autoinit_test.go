package autoinit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/spf13/cobra"
)

func TestHasCompleteLocalAppConfig(t *testing.T) {
	t.Run("returns_false_when_local_config_is_missing", func(t *testing.T) {
		withTempDir(t)

		complete, err := HasCompleteLocalAppConfig()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if complete {
			t.Fatalf("expected missing local config to be incomplete")
		}
	})

	t.Run("returns_false_when_required_app_fields_are_missing", func(t *testing.T) {
		dir := withTempDir(t)
		writeLocalConfig(t, dir, `{
  "organization_id": "org_test",
  "project_id": "proj_test",
  "app_id": "app_test"
}`)

		complete, err := HasCompleteLocalAppConfig()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if complete {
			t.Fatalf("expected config without app_alternate_id to be incomplete")
		}
	})

	t.Run("returns_true_when_local_app_config_is_complete", func(t *testing.T) {
		dir := withTempDir(t)
		writeLocalConfig(t, dir, `{
  "organization_id": "org_test",
  "project_id": "proj_test",
  "app_id": "app_test",
  "app_alternate_id": "app-test"
}`)

		complete, err := HasCompleteLocalAppConfig()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !complete {
			t.Fatalf("expected complete local app config")
		}
	})
}

func TestNeedsLocalAppConfig(t *testing.T) {
	testCases := []struct {
		name     string
		path     []string
		args     []string
		config   func(*cobra.Command)
		expected bool
	}{
		{name: "build", path: []string{"build"}, expected: true},
		{name: "top level pull", path: []string{"pull"}, expected: true},
		{name: "top level push", path: []string{"push"}, expected: true},
		{name: "env pull", path: []string{"env", "pull"}, expected: true},
		{name: "env push", path: []string{"env", "push"}, expected: true},
		{name: "env list", path: []string{"env", "list"}, expected: true},
		{name: "env list versions", path: []string{"env", "list-versions"}, expected: true},
		{name: "env run", path: []string{"env", "run"}, expected: true},
		{name: "env rotate key", path: []string{"env", "rotate-key"}, expected: true},
		{name: "code docker", path: []string{"code", "docker"}, expected: true},
		{name: "project list", path: []string{"project", "list"}, expected: false},
		{name: "init", path: []string{"init"}, expected: false},
		{name: "deploy default", path: []string{"deploy"}, expected: true},
		{name: "deploy explicit id builds local app", path: []string{"deploy"}, args: []string{"depl_test"}, expected: true},
		{
			name: "deploy explicit id with no build",
			path: []string{"deploy"},
			args: []string{"depl_test"},
			config: func(cmd *cobra.Command) {
				cmd.Flags().Bool("no-build", true, "")
				_ = cmd.Flags().Set("no-build", "true")
			},
			expected: false,
		},
		{
			name: "deploy explicit id with apps",
			path: []string{"deploy"},
			args: []string{"depl_test"},
			config: func(cmd *cobra.Command) {
				cmd.Flags().String("apps", "", "")
				_ = cmd.Flags().Set("apps", "app_test")
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := commandForPath(tc.path...)
			if tc.config != nil {
				tc.config(cmd)
			}

			actual := NeedsLocalAppConfig(cmd, tc.args)

			if actual != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestEnsure(t *testing.T) {
	t.Run("does_nothing_when_command_does_not_need_local_app_config", func(t *testing.T) {
		withTempDir(t)
		restore := stubAutoInit(t)
		restore.runInitApp = func(cmd *cobra.Command, args []string) error {
			t.Fatalf("runInitApp should not be called")
			return nil
		}

		err := Ensure(commandForPath("project", "list"), nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("does_nothing_when_local_app_config_is_complete", func(t *testing.T) {
		dir := withTempDir(t)
		writeCompleteLocalConfig(t, dir)
		restore := stubAutoInit(t)
		restore.runInitApp = func(cmd *cobra.Command, args []string) error {
			t.Fatalf("runInitApp should not be called")
			return nil
		}

		err := Ensure(commandForPath("build"), nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("fails_with_guidance_when_noninteractive", func(t *testing.T) {
		withTempDir(t)
		restore := stubAutoInit(t)
		restore.isStdinTerminal = func() bool { return false }
		restore.runInitApp = func(cmd *cobra.Command, args []string) error {
			t.Fatalf("runInitApp should not be called")
			return nil
		}

		err := Ensure(commandForPath("build"), nil)

		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "Run `hx init`") {
			t.Fatalf("expected guidance error, got: %v", err)
		}
	})

	t.Run("fails_with_guidance_when_no_flag_is_set", func(t *testing.T) {
		withTempDir(t)
		restore := stubAutoInit(t)
		restore.runInitApp = func(cmd *cobra.Command, args []string) error {
			t.Fatalf("runInitApp should not be called")
			return nil
		}
		cmd := commandForPath("build")
		cmd.Flags().Bool("no", false, "")
		_ = cmd.Flags().Set("no", "true")

		err := Ensure(cmd, nil)

		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "Run `hx init`") {
			t.Fatalf("expected guidance error, got: %v", err)
		}
	})

	t.Run("fails_with_guidance_for_json_output", func(t *testing.T) {
		withTempDir(t)
		restore := stubAutoInit(t)
		restore.runInitApp = func(cmd *cobra.Command, args []string) error {
			t.Fatalf("runInitApp should not be called")
			return nil
		}
		cmd := commandForPath("build")
		cmd.Flags().String("output", "", "")
		_ = cmd.Flags().Set("output", "json")

		err := Ensure(cmd, nil)

		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "Run `hx init`") {
			t.Fatalf("expected guidance error, got: %v", err)
		}
	})

	t.Run("runs_init_and_revalidates_before_continuing", func(t *testing.T) {
		dir := withTempDir(t)
		restore := stubAutoInit(t)
		called := false
		restore.runInitApp = func(cmd *cobra.Command, args []string) error {
			called = true
			if len(args) != 0 {
				t.Fatalf("expected auto-init to ignore original command args, got %v", args)
			}
			writeCompleteLocalConfig(t, dir)
			return nil
		}

		err := Ensure(commandForPath("build"), []string{"ignored"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatalf("expected runInitApp to be called")
		}
	})

	t.Run("returns_init_error", func(t *testing.T) {
		withTempDir(t)
		restore := stubAutoInit(t)
		restore.runInitApp = func(cmd *cobra.Command, args []string) error {
			return errors.New("init failed")
		}

		err := Ensure(commandForPath("build"), nil)

		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "auto-init failed") {
			t.Fatalf("expected wrapped init error, got: %v", err)
		}
	})
}

type autoInitStubs struct {
	isStdinTerminal func() bool
	runInitApp      func(*cobra.Command, []string) error
}

func stubAutoInit(t *testing.T) *autoInitStubs {
	t.Helper()

	originalIsStdinTerminal := isStdinTerminal
	originalRunInitApp := runInitApp
	originalEnsureAuthenticated := ensureAuthenticated

	stubs := &autoInitStubs{
		isStdinTerminal: func() bool { return true },
		runInitApp:      func(cmd *cobra.Command, args []string) error { return nil },
	}

	isStdinTerminal = func() bool { return stubs.isStdinTerminal() }
	runInitApp = func(cmd *cobra.Command, args []string) error { return stubs.runInitApp(cmd, args) }
	ensureAuthenticated = func() error { return nil }

	t.Cleanup(func() {
		isStdinTerminal = originalIsStdinTerminal
		runInitApp = originalRunInitApp
		ensureAuthenticated = originalEnsureAuthenticated
	})

	return stubs
}

func commandForPath(parts ...string) *cobra.Command {
	root := &cobra.Command{Use: "hyphen"}
	current := root
	for _, part := range parts {
		child := &cobra.Command{Use: part}
		current.AddCommand(child)
		current = child
	}
	return current
}

func withTempDir(t *testing.T) string {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("failed to restore current directory: %v", err)
		}
	})

	return dir
}

func writeCompleteLocalConfig(t *testing.T, dir string) {
	t.Helper()
	writeLocalConfig(t, dir, `{
  "organization_id": "org_test",
  "project_id": "proj_test",
  "app_id": "app_test",
  "app_alternate_id": "app-test"
}`)
}

func writeLocalConfig(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, config.ManifestConfigFile), []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write local config: %v", err)
	}
}
