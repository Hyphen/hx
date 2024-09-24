package zelda

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

type ZeldaServicer interface {
	CreateCode(code Code) (Code, error)
	CreateQRCode(organizationID, codeId string) (QR, error)
	ListDomains(organizationID string, pageSize, pageNum int) ([]DomainInfo, error)
}

type ZeldaService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() *ZeldaService {
	baseUrl := fmt.Sprintf("%s/%s", apiconf.GetBaseApixUrl(), "link")
	return &ZeldaService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (zs *ZeldaService) CreateCode(code Code) (Code, error) {
	url := fmt.Sprintf("%s/codes/", zs.baseUrl)

	payload, err := json.Marshal(code)
	if err != nil {
		return Code{}, errors.Wrap(err, "Failed to marshal request payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return Code{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := zs.httpClient.Do(req)
	if err != nil {
		return Code{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return Code{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Code{}, errors.Wrap(err, "Failed to read response body")
	}

	var createdCode Code
	err = json.Unmarshal(body, &createdCode)
	if err != nil {
		return Code{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return createdCode, nil
}

func (zs *ZeldaService) ListDomains(organizationId string, pageSize, pageNum int) ([]DomainInfo, error) {
	url := fmt.Sprintf("%s/domains/?organizationId=%s&pageSize=%d&pageNum=%d",
		zs.baseUrl, organizationId, pageSize, pageNum)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create request")
	}

	resp, err := zs.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var response struct {
		Data []DomainInfo `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}

func (zs *ZeldaService) CreateQRCode(organizationID, codeId string) (QR, error) {
	url := fmt.Sprintf("%s/codes/%s/qr", zs.baseUrl, codeId)

	// Create the request payload
	payload := struct {
		OrganizationID string `json:"organizationId,omitempty"`
	}{
		OrganizationID: organizationID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return QR{}, errors.Wrap(err, "Failed to marshal request payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return QR{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := zs.httpClient.Do(req)
	if err != nil {
		return QR{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return QR{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return QR{}, errors.Wrap(err, "Failed to read response body")
	}

	var qr QR
	err = json.Unmarshal(body, &qr)
	if err != nil {
		return QR{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return qr, nil
}
