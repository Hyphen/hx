package hyphenapp

import (
	"testing"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentRunLink(t *testing.T) {
	resetAppUrlConfig(t)

	t.Run("uses project and environment alternate IDs", func(t *testing.T) {
		deployment := models.Deployment{
			ID: "depl_123",
			Project: models.ProjectReference{
				ID:          "proj_123",
				AlternateID: "project-alt",
			},
			ProjectEnvironment: models.ProjectEnvironmentReference{
				ID:          "pevr_123",
				AlternateID: "env-alt",
			},
		}

		link := DeploymentRunLink("org_123", &deployment, "run_123")

		assert.Equal(t, "https://app.hyphen.ai/org_123/projects/project-alt/environments/env-alt/runs/run_123", link)
	})

	t.Run("falls back to project and environment IDs", func(t *testing.T) {
		deployment := models.Deployment{
			ID: "depl_123",
			Project: models.ProjectReference{
				ID: "proj_123",
			},
			ProjectEnvironment: models.ProjectEnvironmentReference{
				ID: "pevr_123",
			},
		}

		link := DeploymentRunLink("org_123", &deployment, "run_123")

		assert.Equal(t, "https://app.hyphen.ai/org_123/projects/proj_123/environments/pevr_123/runs/run_123", link)
	})

	t.Run("falls back to the deployment route without project", func(t *testing.T) {
		deployment := models.Deployment{
			ID: "depl_123",
			ProjectEnvironment: models.ProjectEnvironmentReference{
				ID:          "pevr_123",
				AlternateID: "env-alt",
			},
		}

		link := DeploymentRunLink("org_123", &deployment, "run_123")

		assert.Equal(t, "https://app.hyphen.ai/org_123/deploy/depl_123/runs/run_123", link)
	})

	t.Run("falls back to the deployment route without environment", func(t *testing.T) {
		deployment := models.Deployment{
			ID: "depl_123",
			Project: models.ProjectReference{
				ID:          "proj_123",
				AlternateID: "project-alt",
			},
		}

		link := DeploymentRunLink("org_123", &deployment, "run_123")

		assert.Equal(t, "https://app.hyphen.ai/org_123/deploy/depl_123/runs/run_123", link)
	})
}

func TestDeploymentRunLinkForRun(t *testing.T) {
	resetAppUrlConfig(t)

	t.Run("uses run deployment snapshot when selected deployment lacks route metadata", func(t *testing.T) {
		selectedDeployment := models.Deployment{
			ID: "depl_selected",
		}
		run := models.DeploymentRun{
			ID: "run_123",
			DeploymentSnapshot: models.Deployment{
				ID: "depl_snapshot",
				Project: models.ProjectReference{
					ID:          "proj_123",
					AlternateID: "project-alt",
				},
				ProjectEnvironment: models.ProjectEnvironmentReference{
					ID:          "pevr_123",
					AlternateID: "env-alt",
				},
			},
		}

		link := DeploymentRunLinkForRun("org_123", &selectedDeployment, &run)

		assert.Equal(t, "https://app.hyphen.ai/org_123/projects/project-alt/environments/env-alt/runs/run_123", link)
	})

	t.Run("uses selected deployment when run deployment snapshot lacks route metadata", func(t *testing.T) {
		selectedDeployment := models.Deployment{
			ID: "depl_selected",
			Project: models.ProjectReference{
				ID:          "proj_123",
				AlternateID: "project-alt",
			},
			ProjectEnvironment: models.ProjectEnvironmentReference{
				ID:          "pevr_123",
				AlternateID: "env-alt",
			},
		}
		run := models.DeploymentRun{
			ID: "run_123",
			DeploymentSnapshot: models.Deployment{
				ID: "depl_snapshot",
			},
		}

		link := DeploymentRunLinkForRun("org_123", &selectedDeployment, &run)

		assert.Equal(t, "https://app.hyphen.ai/org_123/projects/project-alt/environments/env-alt/runs/run_123", link)
	})
}

func TestDeploymentRunLinkNilInputs(t *testing.T) {
	resetAppUrlConfig(t)

	assert.Empty(t, DeploymentRunLink("org_123", nil, "run_123"))
	assert.Empty(t, DeploymentRunLinkForRun("org_123", &models.Deployment{ID: "depl_123"}, nil))
}

func resetAppUrlConfig(t *testing.T) {
	originalDevFlag := flags.DevFlag
	flags.DevFlag = false
	t.Cleanup(func() { flags.DevFlag = originalDevFlag })
	t.Setenv("HYPHEN_DEV", "false")
	t.Setenv("HYPHEN_Local", "false")
}
