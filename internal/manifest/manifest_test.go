package manifest

import (
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	// Temporarily change ManifestConfigFile for testing
	oldManifestConfigFile := ManifestConfigFile
	ManifestConfigFile = ".test-manifest-key"
	defer func() {
		ManifestConfigFile = oldManifestConfigFile
		os.Remove(".test-manifest-key")
	}()

	t.Run("Successful initialization", func(t *testing.T) {
		m, err := Initialize("org1", "TestProject", "proj1", "test-proj")
		assert.NoError(t, err)
		assert.Equal(t, "TestProject", m.ProjectName)
		assert.Equal(t, "proj1", m.ProjectId)
		assert.Equal(t, "test-proj", m.ProjectAlternateId)
		assert.NotEmpty(t, m.SecretKey)

		// Check if file was created
		_, err = os.Stat(ManifestConfigFile)
		assert.NoError(t, err)
	})

	t.Run("Error creating file", func(t *testing.T) {
		// Set ManifestConfigFile to a path that we can't write to
		ManifestConfigFile = "/root/.test-manifest-key"
		defer func() { ManifestConfigFile = ".test-manifest-key" }()

		_, err := Initialize("org1", "TestProject", "proj1", "test-proj")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error creating file")
	})
}

func TestRestoreFromFile(t *testing.T) {
	t.Run("Successful restore", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test-manifest-*.toml")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		content := `
project_name = "TestProject"
project_id = "proj1"
project_alternate_id = "test-proj"
secret_key = "dGVzdC1zZWNyZXQta2V5"
`
		_, err = tempFile.WriteString(content)
		require.NoError(t, err)
		tempFile.Close()

		m, err := RestoreFromFile(tempFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, "TestProject", m.ProjectName)
		assert.Equal(t, "proj1", m.ProjectId)
		assert.Equal(t, "test-proj", m.ProjectAlternateId)
		assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", m.SecretKey)
	})

	t.Run("File does not exist", func(t *testing.T) {
		_, err := RestoreFromFile("non-existent-file")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "You must init the environment with 'env init'")
	})

	t.Run("Invalid TOML content", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test-manifest-*.toml")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		content := `
invalid toml content
`
		_, err = tempFile.WriteString(content)
		require.NoError(t, err)
		tempFile.Close()

		_, err = RestoreFromFile(tempFile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error decoding TOML file")
	})
}

func TestRestore(t *testing.T) {
	oldManifestConfigFile := ManifestConfigFile
	ManifestConfigFile = ".test-manifest-key"
	defer func() {
		ManifestConfigFile = oldManifestConfigFile
		os.Remove(".test-manifest-key")
	}()

	t.Run("Successful restore", func(t *testing.T) {
		content := `
project_name = "TestProject"
project_id = "proj1"
project_alternate_id = "test-proj"
secret_key = "dGVzdC1zZWNyZXQta2V5"
`
		err := os.WriteFile(ManifestConfigFile, []byte(content), 0644)
		require.NoError(t, err)

		m, err := Restore()
		assert.NoError(t, err)
		assert.Equal(t, "TestProject", m.ProjectName)
		assert.Equal(t, "proj1", m.ProjectId)
		assert.Equal(t, "test-proj", m.ProjectAlternateId)
		assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", m.SecretKey)
	})

	t.Run("File does not exist", func(t *testing.T) {
		os.Remove(ManifestConfigFile)
		_, err := Restore()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "You must init the environment with 'env init'")
	})
}

func TestExists(t *testing.T) {
	oldManifestConfigFile := ManifestConfigFile
	ManifestConfigFile = ".test-manifest-key"
	defer func() {
		ManifestConfigFile = oldManifestConfigFile
		os.Remove(".test-manifest-key")
	}()

	t.Run("File does not exist", func(t *testing.T) {
		os.Remove(ManifestConfigFile)
		assert.False(t, Exists())
	})

	t.Run("File exists", func(t *testing.T) {
		_, err := os.Create(ManifestConfigFile)
		require.NoError(t, err)
		assert.True(t, Exists())
	})
}

func TestGetSecretKey(t *testing.T) {
	m := Manifest{
		SecretKey: "dGVzdC1zZWNyZXQta2V5",
	}

	sk := m.GetSecretKey()
	assert.IsType(t, &secretkey.SecretKey{}, sk)
	assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", sk.Base64())
}