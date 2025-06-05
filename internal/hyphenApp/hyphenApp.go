package hyphenapp

import (
	"fmt"

	"github.com/Hyphen/cli/pkg/apiconf"
)

func OrganizationLink(organizationId string) string {
	return fmt.Sprintf("%s/%s", apiconf.GetBaseAppUrl(), organizationId)
}

func ProjectLink(organizationId, projectAlternateId string) string {
	return fmt.Sprintf("%s/%s/projects/%s", apiconf.GetBaseAppUrl(), organizationId, projectAlternateId)
}

func ApplicationLink(organizationId, projectAlternateId, appAlternateId string) string {
	return fmt.Sprintf("%s/%s/projects/%s/app/%s", apiconf.GetBaseAppUrl(), organizationId, projectAlternateId, appAlternateId)
}

func DeploymentLink(organizationId, deploymentId string) string {
	return fmt.Sprintf("%s/%s/deploy/%s", apiconf.GetBaseAppUrl(), organizationId, deploymentId)
}

func DeploymentRunLink(organizationId, deploymentId, runId string) string {
	return fmt.Sprintf("%s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), organizationId, deploymentId, runId)
}
