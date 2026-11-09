package dockerutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushAuthorizedUsesIsolatedConfigAndPasswordStdin(t *testing.T) {
	directory := t.TempDir()
	script := `#!/bin/sh
printf '%s\n' "$@" >> "$DOCKER_TEST_DIR/args"
printf '%s' "$2" > "$DOCKER_TEST_DIR/config"
if [ "$3" = login ]; then /bin/cat > "$DOCKER_TEST_DIR/stdin"; fi
`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "docker"), []byte(script), 0700))
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DOCKER_TEST_DIR", directory)
	require.NoError(t, PushAuthorized(context.Background(), "local:build", "registry.test/app:build", "registry.test", "user", "credential-sentinel"))
	args, err := os.ReadFile(filepath.Join(directory, "args"))
	require.NoError(t, err)
	input, err := os.ReadFile(filepath.Join(directory, "stdin"))
	require.NoError(t, err)
	config, err := os.ReadFile(filepath.Join(directory, "config"))
	require.NoError(t, err)
	assert.NotContains(t, string(args), "credential-sentinel")
	assert.Equal(t, "credential-sentinel", string(input))
	assert.Equal(t, 3, strings.Count(string(args), "--config"))
	assert.Contains(t, string(args), "--password-stdin")
	_, err = os.Stat(string(config))
	assert.True(t, os.IsNotExist(err))
}

func TestPushAuthorizedSanitizesDockerFailures(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "docker"), []byte("#!/bin/sh\necho credential-sentinel >&2\nexit 1\n"), 0700))
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	err := PushAuthorized(context.Background(), "local:build", "registry.test/app:build", "registry.test", "user", "credential-sentinel")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "credential-sentinel")
}
