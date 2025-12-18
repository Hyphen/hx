package organizations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/httputil"
)

type OrganizationServicer interface {
	ListOrganizations() (models.PaginatedResponse[models.Organization], error)
	GetOrganization(organizationID string) (models.Organization, error)
}

type OrganizationService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() OrganizationService {
	baseUrl := fmt.Sprintf("%s/api/organizations", apiconf.GetBaseApixUrl())
	return OrganizationService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}
func NewTestService(httpClient httputil.Client) OrganizationServicer {
	baseUrl := fmt.Sprintf("%s/api/organizations", apiconf.GetBaseApixUrl())
	return &OrganizationService{
		baseUrl,
		httpClient,
	}
}

func (os *OrganizationService) ListOrganizations() (models.PaginatedResponse[models.Organization], error) {
	url := fmt.Sprintf("%s/", os.baseUrl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.PaginatedResponse[models.Organization]{}, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := os.httpClient.Do(req)
	if err != nil {
		return models.PaginatedResponse[models.Organization]{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.PaginatedResponse[models.Organization]{}, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.PaginatedResponse[models.Organization]{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var organizations models.PaginatedResponse[models.Organization]
	err = json.Unmarshal(body, &organizations)
	if err != nil {
		return models.PaginatedResponse[models.Organization]{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return organizations, nil
}

func (os *OrganizationService) GetOrganization(organizationID string) (models.Organization, error) {
	url := fmt.Sprintf("%s/%s", os.baseUrl, organizationID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.Organization{}, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := os.httpClient.Do(req)
	if err != nil {
		return models.Organization{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Organization{}, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Organization{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var organization models.Organization
	err = json.Unmarshal(body, &organization)
	if err != nil {
		return models.Organization{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return organization, nil
}
