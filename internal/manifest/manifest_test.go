package manifest

import (
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConfigProvider struct {
	configDir string
}

func (m *mockConfigProvider) GetConfigDirectory() string {
	return m.configDir
}

func TestInitialize(t *testing.T) {
	// Temporarily change ManifestConfigFile and ManifestSecretFile for testing
	oldManifestConfigFile := ManifestConfigFile
	oldManifestSecretFile := ManifestSecretFile
	ManifestConfigFile = ".test-manifest-key.json"
	ManifestSecretFile = ".test-manifest-secret-key.json"
	defer func() {
		ManifestConfigFile = oldManifestConfigFile
		ManifestSecretFile = oldManifestSecretFile
		os.Remove(".test-manifest-key.json")
		os.Remove(".test-manifest-secret-key.json")
	}()

	t.Run("Successful initialization", func(t *testing.T) {
		m, err := Initialize("org1", "TestApp", "app1", "test-app")
		assert.NoError(t, err)
		assert.Equal(t, "TestApp", m.AppName)
		assert.Equal(t, "app1", m.AppId)
		assert.Equal(t, "test-app", m.AppAlternateId)
		assert.NotEmpty(t, m.SecretKey)

		// Check if files were created
		_, err = os.Stat(ManifestConfigFile)
		assert.NoError(t, err)
		_, err = os.Stat(ManifestSecretFile)
		assert.NoError(t, err)
	})

	t.Run("Error creating file", func(t *testing.T) {
		// Set ManifestConfigFile to a path that we can't write to
		ManifestConfigFile = "/root/.test-manifest-key.json"
		defer func() { ManifestConfigFile = ".test-manifest-key.json" }()

		_, err := Initialize("org1", "TestApp", "app1", "test-app")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error writing file")
	})
}

func TestRestoreFromFile(t *testing.T) {
	// Set up mock config provider
	mockProvider := &mockConfigProvider{configDir: "/mock/config"}
	SetConfigProvider(mockProvider)
	defer SetConfigProvider(&defaultConfigProvider{})

	t.Run("Successful restore with local files", func(t *testing.T) {
		// Create temporary files for testing
		localConfigFile := createTempFile(t, `{
			"app_name": "TestApp",
			"app_id": "app1",
			"app_alternate_id": "test-app",
			"organisation_id": "org1"
		}`)
		defer os.Remove(localConfigFile.Name())

		localSecretFile := createTempFile(t, `{
			"secret_key": "dGVzdC1zZWNyZXQta2V5"
		}`)
		defer os.Remove(localSecretFile.Name())

		m, err := RestoreFromFile(localConfigFile.Name(), localSecretFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, "TestApp", m.AppName)
		assert.Equal(t, "app1", m.AppId)
		assert.Equal(t, "test-app", m.AppAlternateId)
		assert.Equal(t, "org1", m.OrganisationId)
		assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", m.SecretKey)
	})

	t.Run("No config files exist", func(t *testing.T) {
		_, err := RestoreFromFile("non-existent-config.json", "non-existent-secret.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No valid configuration found")
	})

	t.Run("Invalid JSON content", func(t *testing.T) {
		invalidFile := createTempFile(t, `invalid json content`)
		defer os.Remove(invalidFile.Name())

		_, err := RestoreFromFile(invalidFile.Name(), invalidFile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No valid configuration found (neither global nor local)")
	})
}

func TestRestore(t *testing.T) {
	oldManifestConfigFile := ManifestConfigFile
	oldManifestSecretFile := ManifestSecretFile
	ManifestConfigFile = ".test-manifest-key.json"
	ManifestSecretFile = ".test-manifest-secret-key.json"
	defer func() {
		ManifestConfigFile = oldManifestConfigFile
		ManifestSecretFile = oldManifestSecretFile
		os.Remove(".test-manifest-key.json")
		os.Remove(".test-manifest-secret-key.json")
	}()

	t.Run("Successful restore", func(t *testing.T) {
		configContent := `{
			"app_name": "TestApp",
			"app_id": "app1",
			"app_alternate_id": "test-app",
			"organisation_id": "org1"
		}`
		err := os.WriteFile(ManifestConfigFile, []byte(configContent), 0644)
		require.NoError(t, err)

		secretContent := `{
			"secret_key": "dGVzdC1zZWNyZXQta2V5"
		}`
		err = os.WriteFile(ManifestSecretFile, []byte(secretContent), 0644)
		require.NoError(t, err)

		m, err := Restore()
		assert.NoError(t, err)
		assert.Equal(t, "TestApp", m.AppName)
		assert.Equal(t, "app1", m.AppId)
		assert.Equal(t, "test-app", m.AppAlternateId)
		assert.Equal(t, "org1", m.OrganisationId)
		assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", m.SecretKey)
	})

	t.Run("Files do not exist", func(t *testing.T) {
		os.Remove(ManifestConfigFile)
		os.Remove(ManifestSecretFile)
		_, err := Restore()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No valid configuration found")
	})
}

func TestExists(t *testing.T) {
	oldManifestConfigFile := ManifestConfigFile
	ManifestConfigFile = ".test-manifest-key.json"
	defer func() {
		ManifestConfigFile = oldManifestConfigFile
		os.Remove(".test-manifest-key.json")
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
	ms := ManifestSecret{
		SecretKey: "dGVzdC1zZWNyZXQta2V5",
	}
	m := Manifest{
		ManifestConfig{},
		ms,
	}

	sk := m.GetSecretKey()
	assert.IsType(t, &secretkey.SecretKey{}, sk)
	assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", sk.Base64())
}

func createTempFile(t *testing.T, content string) *os.File {
	tmpfile, err := os.CreateTemp("", "test-manifest-*.json")
	require.NoError(t, err)

	_, err = tmpfile.Write([]byte(content))
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	return tmpfile
}
