package vinz

import (
	"bytes"
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
	SaveKey(organizationID, projectIdOrAlternateId string, key Key) (Key, error)
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

func (vs *VinzService) SaveKey(organizationID, projectIdOrAlternateId string, key Key) (Key, error) {
	url := fmt.Sprintf("%s/%s/%s/key", vs.baseUrl, organizationID, projectIdOrAlternateId)
	// Marshal the key as JSON for the request body
	keyBody, err := json.Marshal(key)
	if err != nil {
		return Key{}, errors.Wrap(err, "Failed to marshal key")
	}

	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewReader(keyBody)))
	if err != nil {
		return Key{}, errors.Wrap(err, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return Key{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
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
