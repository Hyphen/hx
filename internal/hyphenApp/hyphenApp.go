package hyphenapp

import (
	"fmt"

	"github.com/Hyphen/cli/internal/models"
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

func ApplicationBuildLink(organizationId, projectAlternateId, appAlternateId, buildId string) string {
	return fmt.Sprintf("%s/%s/projects/%s/app/%s/builds#%s", apiconf.GetBaseAppUrl(), organizationId, projectAlternateId, appAlternateId, buildId)
}

func DeploymentLink(organizationId, deploymentId string) string {
	return fmt.Sprintf("%s/%s/deploy/%s", apiconf.GetBaseAppUrl(), organizationId, deploymentId)
}

func DeploymentRunLinkForRun(organizationId string, selectedDeployment models.Deployment, run models.DeploymentRun) string {
	deployment := selectedDeployment
	if hasProjectEnvironmentRunPath(run.DeploymentSnapshot) {
		deployment = run.DeploymentSnapshot
	}

	return DeploymentRunLink(organizationId, deployment, run.ID)
}

func DeploymentRunLink(organizationId string, deployment models.Deployment, runId string) string {
	projectSegment := deploymentProjectSegment(deployment)
	environmentSegment := deploymentEnvironmentSegment(deployment)

	if projectSegment != "" && environmentSegment != "" {
		return fmt.Sprintf(
			"%s/%s/projects/%s/environments/%s/runs/%s",
			apiconf.GetBaseAppUrl(),
			organizationId,
			projectSegment,
			environmentSegment,
			runId,
		)
	}

	return fmt.Sprintf("%s/%s/deploy/%s/runs/%s", apiconf.GetBaseAppUrl(), organizationId, deployment.ID, runId)
}

func hasProjectEnvironmentRunPath(deployment models.Deployment) bool {
	return deploymentProjectSegment(deployment) != "" && deploymentEnvironmentSegment(deployment) != ""
}

func deploymentProjectSegment(deployment models.Deployment) string {
	if deployment.Project.AlternateID != "" {
		return deployment.Project.AlternateID
	}

	return deployment.Project.ID
}

func deploymentEnvironmentSegment(deployment models.Deployment) string {
	if deployment.ProjectEnvironment.AlternateID != "" {
		return deployment.ProjectEnvironment.AlternateID
	}

	return deployment.ProjectEnvironment.ID
}
