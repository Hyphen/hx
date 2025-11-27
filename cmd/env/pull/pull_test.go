package pull

import (
	"testing"

	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFilterEnvsByCurrentEnvironments(t *testing.T) {
	t.Run("returns_only_envs_that_exist_in_current_environments", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "development"}},
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "staging"}},
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "production"}},
		}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
			{AlternateID: "production"},
		}

		result := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"development", "production"}, result)
	})

	t.Run("returns_default_even_when_not_in_current_environments", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: nil}, // nil ProjectEnv means "default"
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "development"}},
		}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
		}

		result := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"default", "development"}, result)
	})

	t.Run("filters_out_deleted_environments", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "development"}},
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "deleted-env"}},
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "production"}},
		}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
			{AlternateID: "production"},
		}

		result := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"development", "production"}, result)
		assert.NotContains(t, result, "deleted-env")
	})

	t.Run("returns_empty_slice_when_all_envs_are_deleted", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "deleted-env-1"}},
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "deleted-env-2"}},
		}
		currentEnvironments := []models.Environment{}

		result := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

		assert.Empty(t, result)
	})

	t.Run("returns_empty_slice_when_no_envs_exist", func(t *testing.T) {
		allEnvs := []models.Env{}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
		}

		result := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

		assert.Empty(t, result)
	})

	t.Run("returns_only_default_when_only_default_exists_and_all_others_deleted", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: nil}, // default
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "deleted-env"}},
		}
		currentEnvironments := []models.Environment{}

		result := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"default"}, result)
	})

	t.Run("preserves_order_of_envs", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "production"}},
			{ProjectEnv: nil}, // default
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "development"}},
		}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
			{AlternateID: "production"},
		}

		result := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"production", "default", "development"}, result)
	})
}
