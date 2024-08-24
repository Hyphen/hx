package env

import (
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
)

// GET /organization/{id}/projects/{id}/envs
// PUT /organization/{id}/projects/{id}/environments/{id}/env
// GET /organization/{id}/projects/{id}/environments/{id}/env
// GET /organization/{id}/projects/{id}/environments/{id}/env/versions
// GET /organization/{id}/projects/{id}/environments/{id}/env/versions/{id}
// DELETE /organization/{id}/projects/{id}/environments/{id}/env

type EnvServicer interface {
	UploadEnvVariable(organizationID, env, projectID string, envData EnvironmentVarsData) error
	GetEncryptedVariables(env, appID string) (EnvironmentVarsData, error)
	ListEnvironments(appID string, pageSize, pageNum int) ([]EnvironmentInformation, error)
}

type EnvService struct {
	baseUrl      string
	oauthService oauth.OAuthServiceInterface
}

func (es *EnvService) UploadEnvVariable(organizationID, env, appID string, envData EnvironmentVarsData) error {
	_, err := es.oauthService.GetValidToken()
	if err != nil {
		return errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}
	return nil
}
