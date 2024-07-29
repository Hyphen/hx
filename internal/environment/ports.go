package environment

import "github.com/Hyphen/cli/internal/environment/envvars"

type Repository interface {
	UploadEnvVariable(env, appID string, envData envvars.EnvironmentVarsData) error
	GetEncryptedVariables(env, appID string) (string, error)
	ListEnvironments(appID string, pageSize, pageNum int) ([]envvars.EnvironmentInformation, error)
	Initialize(apiName, apiId string) error
}
