package env

//
// import (
// 	"net/http"
//
// 	"github.com/Hyphen/cli/internal/manifest"
// 	"github.com/Hyphen/cli/internal/oauth"
// 	"github.com/Hyphen/cli/pkg/errors"
// )
//
// type EnvServicer interface {
// 	UploadEnvVariable(organizationID, env, projectID string, envData EnvInformation) error
// 	GetEncryptedVariables(env, projectID string) (EnvInformation, error)
// 	ListEnvironments(projectID string, pageSize, pageNum int) ([]EnvironmentInformation, error)
// }
//
// type EnvService struct {
// 	baseUrl      string
// 	oauthService oauth.OAuthServicer
// 	manifest     manifest.Manifest
// 	httpClient   *http.Client
// }
//
// func (es *EnvService) UploadEnvVariable(organizationID, env, projectID string, envData EnvInformation) error {
// 	_, err := es.oauthService.GetValidToken()
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
// 	}
//
// 	encryptedEnvData, err := envData.EncryptData(es.manifest.GetSecretKey())
// 	return nil
// }
//
// func (es *EnvService) GetEnv(organizationID, projectID, env string) (EnvInformation, error) {
// 	env, err := GetEnvName(env)
// 	if err != nil {
// 		return EnvInformation{}, err
// 	}
//
// 	_, err = es.oauthService.GetValidToken()
// 	if err != nil {
// 		return EnvInformation{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
// 	}
//
// 	//TODO llamar a la api
// 	return EnvInformation{}, nil
// }
