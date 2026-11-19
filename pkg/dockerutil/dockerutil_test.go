package dockerutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDocker(t *testing.T) {
	for _, scenario := range []string{"missing CLI", "unavailable engine", "ready"} {
		t.Run(scenario, func(t *testing.T) {
			directory := t.TempDir()
			t.Setenv("PATH", directory)
			if scenario != "missing CLI" {
				script := "#!/bin/sh\n[ \"$1\" = info ] || exit 2\n"
				if scenario == "unavailable engine" {
					script += "echo secret-sentinel >&2\nexit 1\n"
				}
				require.NoError(t, os.WriteFile(filepath.Join(directory, "docker"), []byte(script), 0700))
			}
			err := CheckDocker(context.Background())
			if scenario == "ready" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			if scenario == "missing CLI" {
				assert.Contains(t, err.Error(), "Docker CLI")
				assert.Contains(t, err.Error(), "PATH")
			} else {
				assert.Contains(t, err.Error(), "requires a running Docker engine")
				assert.Contains(t, err.Error(), "start Docker Desktop")
				assert.Contains(t, err.Error(), "context and permissions")
			}
			assert.NotContains(t, err.Error(), "secret-sentinel")
		})
	}
}

func TestCheckDockerPreservesContextErrors(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "docker"), []byte("#!/bin/sh\nexec /bin/sleep 10\n"), 0700))
	t.Setenv("PATH", directory)
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(want.Error(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if want == context.Canceled {
				cancel()
			}
			assert.ErrorIs(t, CheckDocker(ctx), want)
		})
	}
}

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

func TestPushAuthorizedPreservesContextErrors(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "docker"), []byte("#!/bin/sh\nexec /bin/sleep 10\n"), 0700))
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tc := range []struct {
		name string
		want error
	}{
		{"canceled", context.Canceled},
		{"deadline", context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if tc.want == context.Canceled {
				cancel()
			}
			err := PushAuthorized(ctx, "local:build", "registry.test/app:build", "registry.test", "user", "credential-sentinel")
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPushAuthorizedDigest(t *testing.T) {
	directory := t.TempDir()
	script := "#!/bin/sh\nif [ \"$3\" = push ]; then echo 'build: digest: sha256:" + strings.Repeat("a", 64) + " size: 123'; fi\n"
	require.NoError(t, os.WriteFile(filepath.Join(directory, "docker"), []byte(script), 0700))
	t.Setenv("PATH", directory)
	digest, err := PushAuthorizedDigest(context.Background(), "local:build", "registry.test/app:build", "registry.test", "user", "credential-sentinel")
	require.NoError(t, err)
	require.Equal(t, "sha256:"+strings.Repeat("a", 64), digest)
}
