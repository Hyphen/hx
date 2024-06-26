package environment

type Repository interface {
	UploadEnvVariable(env, encryptedVars string) error
	GetEncryptedVariables(env string) (string, error)
}
