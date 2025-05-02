package vinz

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type VinzServicer interface {
	GetKey(organizationID, projectIdOrAlternateId string) (Key, error)
}

type VinzService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() *VinzService {
	baseUrl := apiconf.GetBaseVinzUrl()
	return &VinzService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (vs *VinzService) GetKey(organizationID, projectIdOrAlternateId string) (Key, error) {
	url := fmt.Sprintf("%s/%s/%s/key", vs.baseUrl, organizationID, projectIdOrAlternateId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Key{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return Key{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Key{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Key{}, errors.Wrap(err, "Failed to read response body")
	}

	var keyResponse KeyResponse
	if err := json.Unmarshal(body, &keyResponse); err != nil {
		return Key{}, errors.Wrap(err, "Failed to unmarshal response body")
	}
	return keyResponse.Key, nil
}
