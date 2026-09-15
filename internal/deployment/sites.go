package deployment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/errors"
)

type patchDeploymentSite struct {
	App                deploymentReference `json:"app"`
	DeploymentSettings models.SiteSettings `json:"deploymentSettings"`
}

func newDeploymentSite(appID, integrationID string) patchDeploymentSite {
	var site patchDeploymentSite
	site.App.ID = appID
	site.DeploymentSettings.Targets = []models.SiteTarget{{ID: integrationID}}
	return site
}

func (ds *DeploymentService) AddSitesToDeployment(organizationID, deploymentID string, appIDs []string, integrationID string) (*models.Deployment, error) {
	current, err := ds.GetDeployment(organizationID, deploymentID)
	if err != nil {
		return nil, err
	}
	if len(appIDs) == 0 {
		return current, nil
	}
	if integrationID == "" {
		return nil, fmt.Errorf("a HyphenCloud integration is required to add sites")
	}
	sites := make([]patchDeploymentSite, 0, len(current.Sites)+len(appIDs))
	for _, existing := range current.Sites {
		var site patchDeploymentSite
		site.App.ID = existing.App.ID
		site.DeploymentSettings = existing.DeploymentSettings
		sites = append(sites, site)
	}
	for _, id := range appIDs {
		sites = append(sites, newDeploymentSite(id, integrationID))
	}
	data, err := json.Marshal(map[string]any{"sites": sites})
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/organizations/%s/deployments/%s", ds.baseUrl, url.PathEscape(organizationID), url.PathEscape(deploymentID))
	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(resp)
	}
	return ds.GetDeployment(organizationID, deploymentID)
}
