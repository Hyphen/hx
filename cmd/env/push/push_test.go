package push

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func getTestSecret(secretKeyId int64) models.Secret {
	nonBase64SecretValue := "test-secret-test-secret-test-secret-test-secret-test-secret-test-secret"
	base64SecretValue := base64.StdEncoding.EncodeToString([]byte(nonBase64SecretValue))
	secret := models.NewSecret(base64SecretValue)
	secret.SecretKeyId = secretKeyId
	return secret
}

func TestPushEnv(t *testing.T) {
	t.Run("uses_project_secret_key_id_for_first_push", func(t *testing.T) {
		withEnvFile(t, "staging", "KEY=value")

		mockEnvService := env.NewMockEnvService()
		mockDB := new(database.MockDatabase)
		svc := newService(mockEnvService, mockDB, nil)

		theProjectId := "project-123"
		theAppId := "app-456"
		theEnvName := "staging"
		var theProjectSecretKeyId int64 = 99999

		cfg := config.Config{
			ProjectId: &theProjectId,
			AppId:     &theAppId,
		}
		secret := getTestSecret(theProjectSecretKeyId)

		// Local env doesn't exist
		mockDB.On("GetSecret", mock.Anything).Return(database.Secret{}, false)

		mockEnvService.On("PutEnvironmentEnv", "org-1", theAppId, theEnvName, theProjectSecretKeyId, mock.Anything).
			Return(nil).Once()

		result := svc.pushEnv("org-1", theEnvName, theAppId, secret, cfg, nil)

		assert.NoError(t, result.err)
		assert.False(t, result.skipped)
		assert.True(t, result.hasUpdate)
		assert.Equal(t, "KEY=value", result.update.Data)
		assert.Equal(t, 1, result.update.Version)
		mockEnvService.AssertNumberOfCalls(t, "PutEnvironmentEnv", 1)
		mockEnvService.AssertCalled(t, "PutEnvironmentEnv", "org-1", theAppId, theEnvName, theProjectSecretKeyId, mock.Anything)
	})

	t.Run("uses_cloud_secret_key_id_when_env_exists_in_cloud", func(t *testing.T) {
		withEnvFile(t, "production", "KEY=value")

		mockEnvService := env.NewMockEnvService()
		mockDB := new(database.MockDatabase)
		svc := newService(mockEnvService, mockDB, nil)

		theProjectId := "project-123"
		theAppId := "app-456"
		theEnvName := "production"
		var theProjectSecretKeyId int64 = 11111
		var theCloudSecretKeyId int64 = 22222
		theCloudVersion := 5

		cfg := config.Config{
			ProjectId: &theProjectId,
			AppId:     &theAppId,
		}
		secret := getTestSecret(theProjectSecretKeyId)

		// Local env exists
		mockDB.On("GetSecret", mock.Anything).Return(database.Secret{Version: 3, Hash: "different-hash"}, true)

		// Cloud env exists with different secret key
		cloudEnv := models.Env{
			SecretKeyID: &theCloudSecretKeyId,
			Version:     &theCloudVersion,
		}

		// Expect PutEnvironmentEnv to be called with the CLOUD's secret key ID
		mockEnvService.On("PutEnvironmentEnv", "org-1", theAppId, theEnvName, theCloudSecretKeyId, mock.Anything).
			Return(nil)

		result := svc.pushEnv("org-1", theEnvName, theAppId, secret, cfg, &cloudEnv)

		assert.NoError(t, result.err)
		assert.False(t, result.skipped)
		assert.True(t, result.hasUpdate)
		mockEnvService.AssertCalled(t, "PutEnvironmentEnv", "org-1", theAppId, theEnvName, theCloudSecretKeyId, mock.Anything)
		mockEnvService.AssertNotCalled(t, "GetEnvironmentEnv", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("returns_error_when_put_fails_with_non_secretKeyId_error", func(t *testing.T) {
		withEnvFile(t, "staging", "KEY=value")

		mockEnvService := env.NewMockEnvService()
		mockDB := new(database.MockDatabase)
		svc := newService(mockEnvService, mockDB, nil)

		theProjectId := "project-123"
		theAppId := "app-456"
		theEnvName := "staging"
		var theProjectSecretKeyId int64 = 99999

		cfg := config.Config{
			ProjectId: &theProjectId,
			AppId:     &theAppId,
		}
		secret := getTestSecret(theProjectSecretKeyId)

		// Local env doesn't exist
		mockDB.On("GetSecret", mock.Anything).Return(database.Secret{}, false)

		// PutEnvironmentEnv fails with a different error
		mockEnvService.On("PutEnvironmentEnv", "org-1", theAppId, theEnvName, theProjectSecretKeyId, mock.Anything).
			Return(errors.Wrapf(errors.ErrUnauthorized, "unauthorized")).Once()

		result := svc.pushEnv("org-1", theEnvName, theAppId, secret, cfg, nil)

		assert.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "failed to update cloud staging environment")
		assert.False(t, result.skipped)
		mockEnvService.AssertNumberOfCalls(t, "PutEnvironmentEnv", 1)
	})

	t.Run("skips_unchanged_env_when_cloud_secret_matches_project_secret", func(t *testing.T) {
		plainData := "KEY=value"
		withEnvFile(t, "staging", plainData)

		mockEnvService := env.NewMockEnvService()
		mockDB := new(database.MockDatabase)
		svc := newService(mockEnvService, mockDB, nil)

		theProjectId := "project-123"
		theAppId := "app-456"
		theEnvName := "staging"
		var theProjectSecretKeyId int64 = 99999
		theVersion := 3

		cfg := config.Config{
			ProjectId: &theProjectId,
			AppId:     &theAppId,
		}
		secret := getTestSecret(theProjectSecretKeyId)

		mockDB.On("GetSecret", mock.Anything).Return(database.Secret{Version: 3, Hash: models.HashData(plainData)}, true)

		cloudEnv := models.Env{
			SecretKeyID: &theProjectSecretKeyId,
			Version:     &theVersion,
		}

		result := svc.pushEnv("org-1", theEnvName, theAppId, secret, cfg, &cloudEnv)

		assert.NoError(t, result.err)
		assert.True(t, result.skipped)
		assert.False(t, result.hasUpdate)
		mockEnvService.AssertNotCalled(t, "PutEnvironmentEnv", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func withEnvFile(t *testing.T, envName, contents string) {
	t.Helper()

	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() { assert.NoError(t, os.Chdir(originalDir)) })

	envFileName, err := env.GetFileName(envName)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(envFileName, []byte(contents), 0644))
}
