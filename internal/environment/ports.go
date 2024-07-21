package environment

import "github.com/Hyphen/cli/internal/environment/envvars"

type Repository interface {
	UploadEnvVariable(env, appID string, envData envvars.EnvironmentVarsData) error
	GetEncryptedVariables(env, appID string) (string, error)
	Initialize(apiName, apiId string) error
}
