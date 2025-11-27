package push

import (
	"encoding/base64"
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

func TestPutEnv(t *testing.T) {
	t.Run("retries_with_project_secret_key_id_when_put_fails_with_secretKeyId_error", func(t *testing.T) {
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

		envData := models.Env{Data: "KEY=value"}
		encryptedData, _ := envData.EncryptData(secret)
		envData.Data = encryptedData

		// Local env doesn't exist
		mockDB.On("GetSecret", mock.Anything).Return(database.Secret{}, false)

		// First PutEnvironmentEnv fails with the specific error
		mockEnvService.On("PutEnvironmentEnv", "org-1", theAppId, theEnvName, int64(0), mock.Anything).
			Return(errors.Wrapf(errors.ErrBadRequest, "bad request: querystring/secretKeyId must be >= 1")).Once()

		// Retry with project secret key ID succeeds
		mockEnvService.On("PutEnvironmentEnv", "org-1", theAppId, theEnvName, theProjectSecretKeyId, mock.Anything).
			Return(nil).Once()

		// UpsertSecret for local storage
		mockDB.On("UpsertSecret", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		var skippedEnvs []string
		err, skipped := svc.putEnv("org-1", theEnvName, theAppId, envData, secret, cfg, &skippedEnvs)

		assert.NoError(t, err)
		assert.False(t, skipped)
		mockEnvService.AssertNumberOfCalls(t, "PutEnvironmentEnv", 2)
	})

	t.Run("uses_cloud_secret_key_id_when_env_exists_in_cloud", func(t *testing.T) {
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

		envData := models.Env{Data: "KEY=value"}
		encryptedData, _ := envData.EncryptData(secret)
		envData.Data = encryptedData

		// Local env exists
		mockDB.On("GetSecret", mock.Anything).Return(database.Secret{Version: 3, Hash: "different-hash"}, true)

		// Cloud env exists with different secret key
		cloudEnv := models.Env{
			SecretKeyID: &theCloudSecretKeyId,
			Version:     &theCloudVersion,
		}
		mockEnvService.On("GetEnvironmentEnv", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(cloudEnv, nil)

		// Expect PutEnvironmentEnv to be called with the CLOUD's secret key ID
		mockEnvService.On("PutEnvironmentEnv", "org-1", theAppId, theEnvName, theCloudSecretKeyId, mock.Anything).
			Return(nil)

		// UpsertSecret for local storage
		mockDB.On("UpsertSecret", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		var skippedEnvs []string
		err, skipped := svc.putEnv("org-1", theEnvName, theAppId, envData, secret, cfg, &skippedEnvs)

		assert.NoError(t, err)
		assert.False(t, skipped)
		mockEnvService.AssertCalled(t, "PutEnvironmentEnv", "org-1", theAppId, theEnvName, theCloudSecretKeyId, mock.Anything)
	})

	t.Run("returns_error_when_put_fails_with_non_secretKeyId_error", func(t *testing.T) {
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

		envData := models.Env{Data: "KEY=value"}
		encryptedData, _ := envData.EncryptData(secret)
		envData.Data = encryptedData

		// Local env doesn't exist
		mockDB.On("GetSecret", mock.Anything).Return(database.Secret{}, false)

		// PutEnvironmentEnv fails with a different error
		mockEnvService.On("PutEnvironmentEnv", "org-1", theAppId, theEnvName, int64(0), mock.Anything).
			Return(errors.Wrapf(errors.ErrUnauthorized, "unauthorized")).Once()

		var skippedEnvs []string
		err, skipped := svc.putEnv("org-1", theEnvName, theAppId, envData, secret, cfg, &skippedEnvs)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update cloud staging environment")
		assert.False(t, skipped)
		mockEnvService.AssertNumberOfCalls(t, "PutEnvironmentEnv", 1)
	})
}
