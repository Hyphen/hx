package manifest

// import (
// 	"os"
// 	"testing"

// 	"github.com/Hyphen/cli/pkg/fsutil"
// 	"github.com/stretchr/testify/assert"
// )

// type mockConfigProvider struct {
// 	configDir string
// }

// func (m *mockConfigProvider) GetConfigDirectory() string {
// 	return m.configDir
// }

// type mockFileSystem struct {
// 	fsutil.FileSystem
// 	files map[string][]byte
// }

// func newMockFileSystem() *mockFileSystem {
// 	return &mockFileSystem{
// 		files: make(map[string][]byte),
// 	}
// }

// func (m *mockFileSystem) ReadFile(filename string) ([]byte, error) {
// 	if data, ok := m.files[filename]; ok {
// 		return data, nil
// 	}
// 	return nil, os.ErrNotExist
// }

// func (m *mockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
// 	m.files[filename] = data
// 	return nil
// }

// func (m *mockFileSystem) Create(name string) (*os.File, error) {
// 	m.files[name] = []byte{}
// 	return nil, nil // Return a proper *os.File if needed
// }

// func (m *mockFileSystem) Stat(name string) (os.FileInfo, error) {
// 	if _, ok := m.files[name]; ok {
// 		return nil, nil // Return a non-nil FileInfo if needed
// 	}
// 	return nil, os.ErrNotExist
// }

// func (m *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
// 	// For simplicity, just pretend the directory was created
// 	return nil
// }

// func TestInitialize(t *testing.T) {
// 	oldFS := FS
// 	mockFS := newMockFileSystem()
// 	FS = mockFS
// 	defer func() { FS = oldFS }()

// 	// Temporarily change ManifestConfigFile and ManifestSecretFile for testing
// 	oldManifestConfigFile := ManifestConfigFile
// 	oldManifestSecretFile := ManifestSecretFile
// 	ManifestConfigFile = ".test-manifest-key.json"
// 	ManifestSecretFile = ".test-manifest-secret-key.json"
// 	defer func() {
// 		ManifestConfigFile = oldManifestConfigFile
// 		ManifestSecretFile = oldManifestSecretFile
// 	}()

// 	t.Run("Successful initialization", func(t *testing.T) {
// 		mc := Config{
// 			AppName:        stringPtr("TestApp"),
// 			AppId:          stringPtr("app1"),
// 			AppAlternateId: stringPtr("test-app"),
// 			OrganizationId: "org1",
// 		}
// 		m, err := Initialize(mc, ManifestSecretFile, ManifestConfigFile)
// 		assert.NoError(t, err)
// 		assert.NotNil(t, m.AppName)
// 		assert.Equal(t, "TestApp", *m.AppName)
// 		assert.NotNil(t, m.AppId)
// 		assert.Equal(t, "app1", *m.AppId)
// 		assert.NotNil(t, m.AppAlternateId)
// 		assert.Equal(t, "test-app", *m.AppAlternateId)
// 		assert.NotEmpty(t, m.SecretKey)

// 		// Check if files were created
// 		_, err = mockFS.Stat(ManifestConfigFile)
// 		assert.NoError(t, err)
// 		_, err = mockFS.Stat(ManifestSecretFile)
// 		assert.NoError(t, err)
// 	})
// }

// func TestRestoreFromFile(t *testing.T) {
// 	oldFS := FS
// 	mockFS := newMockFileSystem()
// 	FS = mockFS
// 	defer func() { FS = oldFS }()

// 	t.Run("Successful restore with local files", func(t *testing.T) {
// 		// Create mock files for testing
// 		localConfigFile := ".test-local-config.json"
// 		mockFS.WriteFile(localConfigFile, []byte(`{
// 			"app_name": "TestApp",
// 			"app_id": "app1",
// 			"app_alternate_id": "test-app",
// 			"organization_id": "org1"
// 		}`), 0644)

// 		localSecretFile := ".test-local-secret.json"
// 		mockFS.WriteFile(localSecretFile, []byte(`{
// 			"secret_key": "dGVzdC1zZWNyZXQta2V5"
// 		}`), 0644)

// 		m, err := RestoreFromFile(localConfigFile, localSecretFile)
// 		assert.NoError(t, err)
// 		assert.NotNil(t, m.AppName)
// 		assert.Equal(t, "TestApp", *m.AppName)
// 		assert.NotNil(t, m.AppId)
// 		assert.Equal(t, "app1", *m.AppId)
// 		assert.NotNil(t, m.AppAlternateId)
// 		assert.Equal(t, "test-app", *m.AppAlternateId)
// 		assert.Equal(t, "org1", m.OrganizationId)
// 		assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", m.SecretKey)
// 	})

// 	t.Run("No config files exist", func(t *testing.T) {
// 		_, err := RestoreFromFile("non-existent-config.json", "non-existent-secret.json")
// 		assert.Error(t, err)
// 		assert.Contains(t, err.Error(), "No valid .hx found (neither global nor local). Please authenticate using `hx auth`")
// 	})

// 	t.Run("Invalid JSON content", func(t *testing.T) {
// 		invalidFile := ".test-invalid.json"
// 		mockFS.WriteFile(invalidFile, []byte(`invalid json content`), 0644)

// 		_, err := RestoreFromFile(invalidFile, invalidFile)
// 		assert.Error(t, err)
// 		assert.Contains(t, err.Error(), "Error decoding JSON file")
// 	})
// }

// func TestRestore(t *testing.T) {
// 	oldFS := FS
// 	mockFS := newMockFileSystem()
// 	FS = mockFS
// 	defer func() { FS = oldFS }()

// 	oldManifestConfigFile := ManifestConfigFile
// 	oldManifestSecretFile := ManifestSecretFile
// 	ManifestConfigFile = ".test-manifest-key.json"
// 	ManifestSecretFile = ".test-manifest-secret-key.json"
// 	defer func() {
// 		ManifestConfigFile = oldManifestConfigFile
// 		ManifestSecretFile = oldManifestSecretFile
// 	}()

// 	t.Run("Successful restore", func(t *testing.T) {
// 		configContent := `{
// 			"app_name": "TestApp",
// 			"app_id": "app1",
// 			"app_alternate_id": "test-app",
// 			"organization_id": "org1"
// 		}`
// 		mockFS.WriteFile(ManifestConfigFile, []byte(configContent), 0644)

// 		secretContent := `{
// 			"secret_key": "dGVzdC1zZWNyZXQta2V5"
// 		}`
// 		mockFS.WriteFile(ManifestSecretFile, []byte(secretContent), 0644)

// 		m, err := Restore()
// 		assert.NoError(t, err)
// 		assert.NotNil(t, m.AppName)
// 		assert.Equal(t, "TestApp", *m.AppName)
// 		assert.NotNil(t, m.AppId)
// 		assert.Equal(t, "app1", *m.AppId)
// 		assert.NotNil(t, m.AppAlternateId)
// 		assert.Equal(t, "test-app", *m.AppAlternateId)
// 		assert.Equal(t, "org1", m.OrganizationId)
// 		assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", m.SecretKey)
// 	})

// 	t.Run("Files do not exist", func(t *testing.T) {
// 		delete(mockFS.files, ManifestConfigFile)
// 		delete(mockFS.files, ManifestSecretFile)
// 		_, err := Restore()
// 		assert.Error(t, err)
// 		assert.Contains(t, err.Error(), "No valid .hx found (neither global nor local). Please authenticate using `hx auth`")
// 	})
// }

// func TestExistsLocal(t *testing.T) {
// 	oldFS := FS
// 	mockFS := newMockFileSystem()
// 	FS = mockFS
// 	defer func() { FS = oldFS }()

// 	oldManifestConfigFile := ManifestConfigFile
// 	ManifestConfigFile = ".test-manifest-key.json"
// 	defer func() {
// 		ManifestConfigFile = oldManifestConfigFile
// 	}()

// 	t.Run("File does not exist", func(t *testing.T) {
// 		delete(mockFS.files, ManifestConfigFile)
// 		assert.False(t, ExistsLocal())
// 	})

// 	t.Run("File exists", func(t *testing.T) {
// 		mockFS.WriteFile(ManifestConfigFile, []byte("test content"), 0644)
// 		assert.True(t, ExistsLocal())
// 	})
// }

// func TestGetSecretKey(t *testing.T) {
// 	ms := Secret{
// 		SecretKey: "dGVzdC1zZWNyZXQta2V5",
// 	}
// 	m := Manifest{
// 		Config: Config{},
// 		Secret: ms,
// 	}

// 	sk := m.GetSecretKey()
// 	assert.IsType(t, &secretkey.SecretKey{}, sk)
// 	assert.Equal(t, "dGVzdC1zZWNyZXQta2V5", sk.Base64())
// }

// func stringPtr(s string) *string {
// 	return &s
// }
