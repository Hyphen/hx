package pull

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestFindMissingEnvironments(t *testing.T) {
	t.Run("returns_environments_that_exist_in_current_but_not_in_allEnvs", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "development"}},
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "production"}},
		}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
			{AlternateID: "production"},
			{AlternateID: "staging"},
			{AlternateID: "newenv"},
		}

		result := findMissingEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"staging", "newenv"}, result)
	})

	t.Run("returns_empty_slice_when_all_environments_have_env_files", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "development"}},
			{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "production"}},
		}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
			{AlternateID: "production"},
		}

		result := findMissingEnvironments(allEnvs, currentEnvironments)

		assert.Empty(t, result)
	})

	t.Run("returns_all_environments_when_no_env_files_exist", func(t *testing.T) {
		allEnvs := []models.Env{}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
			{AlternateID: "production"},
		}

		result := findMissingEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"development", "production"}, result)
	})

	t.Run("handles_default_env_in_allEnvs", func(t *testing.T) {
		allEnvs := []models.Env{
			{ProjectEnv: nil}, // default
		}
		currentEnvironments := []models.Environment{
			{AlternateID: "development"},
		}

		result := findMissingEnvironments(allEnvs, currentEnvironments)

		assert.Equal(t, []string{"development"}, result)
	})
}

func TestCreateEmptyEnvFile(t *testing.T) {
	t.Run("creates_empty_env_file_for_environment", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		t.Cleanup(func() { os.Chdir(originalDir) })

		err := createEmptyEnvFile("staging", false)

		assert.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(tempDir, ".env.staging"))
		assert.NoError(t, err)
		assert.Equal(t, "", string(content))
	})

	t.Run("returns_error_when_file_exists_and_force_is_false", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		t.Cleanup(func() { os.Chdir(originalDir) })

		// Create existing file
		os.WriteFile(filepath.Join(tempDir, ".env.staging"), []byte("existing"), 0644)

		err := createEmptyEnvFile("staging", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("overwrites_file_when_force_is_true", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		t.Cleanup(func() { os.Chdir(originalDir) })

		// Create existing file with content
		os.WriteFile(filepath.Join(tempDir, ".env.staging"), []byte("existing content"), 0644)

		err := createEmptyEnvFile("staging", true)

		assert.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(tempDir, ".env.staging"))
		assert.NoError(t, err)
		assert.Equal(t, "", string(content))
	})

	t.Run("creates_dot_env_file_for_default_environment", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		t.Cleanup(func() { os.Chdir(originalDir) })

		err := createEmptyEnvFile("default", false)

		assert.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(tempDir, ".env"))
		assert.NoError(t, err)
		assert.Equal(t, "", string(content))
	})
}

func TestGetAllEnvsAndDecryptIntoFiles(t *testing.T) {
	t.Run("creates_empty_files_for_environments_missing_from_ListEnvs", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		t.Cleanup(func() { os.Chdir(originalDir) })

		// Initialize printer for the test
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		mockEnvService := env.NewMockEnvService()
		mockDB := new(database.MockDatabase)
		svc := newService(mockEnvService, mockDB, nil)

		theProjectId := "project-123"
		theAppId := "app-456"
		theOrgId := "org-789"

		cfg := config.Config{
			ProjectId: &theProjectId,
			AppId:     &theAppId,
		}

		// ListEnvs returns no env files (empty)
		mockEnvService.On("ListEnvs", theOrgId, theAppId, 100, 1).
			Return([]models.Env{}, nil)

		// ListEnvironments returns two environments
		mockEnvService.On("ListEnvironments", theOrgId, theProjectId, 100, 1).
			Return([]models.Environment{
				{AlternateID: "staging"},
				{AlternateID: "production"},
			}, nil)

		secret := models.Secret{}
		pulledEnvs, err := svc.getAllEnvsAndDecryptIntoFiles(theOrgId, theAppId, theProjectId, secret, cfg, false)

		assert.NoError(t, err)
		assert.Contains(t, pulledEnvs, "staging")
		assert.Contains(t, pulledEnvs, "production")

		// Verify empty files were created
		_, err = os.Stat(filepath.Join(tempDir, ".env.staging"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(tempDir, ".env.production"))
		assert.NoError(t, err)
	})

	t.Run("does_not_create_empty_files_for_environments_that_have_env_files", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		t.Cleanup(func() { os.Chdir(originalDir) })

		// Initialize printer for the test
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		mockEnvService := env.NewMockEnvService()
		mockDB := new(database.MockDatabase)
		svc := newService(mockEnvService, mockDB, nil)

		theProjectId := "project-123"
		theAppId := "app-456"
		theOrgId := "org-789"

		cfg := config.Config{
			ProjectId: &theProjectId,
			AppId:     &theAppId,
		}

		// ListEnvs returns env files for both environments
		mockEnvService.On("ListEnvs", theOrgId, theAppId, 100, 1).
			Return([]models.Env{
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "staging"}},
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "production"}},
			}, nil)

		// ListEnvironments returns the same environments
		mockEnvService.On("ListEnvironments", theOrgId, theProjectId, 100, 1).
			Return([]models.Environment{
				{AlternateID: "staging"},
				{AlternateID: "production"},
			}, nil)

		// Mock the GetEnvironmentEnv calls for saveDecryptedEnvIntoFile
		mockEnvService.On("GetEnvironmentEnv", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(models.Env{}, assert.AnError)

		secret := models.Secret{}
		pulledEnvs, err := svc.getAllEnvsAndDecryptIntoFiles(theOrgId, theAppId, theProjectId, secret, cfg, false)

		assert.NoError(t, err)
		// No envs should be pulled because GetEnvironmentEnv fails
		assert.Empty(t, pulledEnvs)

		// Verify findMissingEnvironments returns empty (no missing envs)
		missingEnvs := findMissingEnvironments(
			[]models.Env{
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "staging"}},
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "production"}},
			},
			[]models.Environment{
				{AlternateID: "staging"},
				{AlternateID: "production"},
			},
		)
		assert.Empty(t, missingEnvs)
	})

	t.Run("filters_out_deleted_environments_from_ListEnvs", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		t.Cleanup(func() { os.Chdir(originalDir) })

		// Initialize printer for the test
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		mockEnvService := env.NewMockEnvService()
		mockDB := new(database.MockDatabase)
		svc := newService(mockEnvService, mockDB, nil)

		theProjectId := "project-123"
		theAppId := "app-456"
		theOrgId := "org-789"

		cfg := config.Config{
			ProjectId: &theProjectId,
			AppId:     &theAppId,
		}

		// ListEnvs returns env files including a deleted one
		mockEnvService.On("ListEnvs", theOrgId, theAppId, 100, 1).
			Return([]models.Env{
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "staging"}},
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "deleted-env"}},
			}, nil)

		// ListEnvironments only returns staging (deleted-env was deleted)
		mockEnvService.On("ListEnvironments", theOrgId, theProjectId, 100, 1).
			Return([]models.Environment{
				{AlternateID: "staging"},
			}, nil)

		// Mock the GetEnvironmentEnv call - it will fail but that's ok for this test
		mockEnvService.On("GetEnvironmentEnv", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(models.Env{}, assert.AnError)

		secret := models.Secret{}
		_, err := svc.getAllEnvsAndDecryptIntoFiles(theOrgId, theAppId, theProjectId, secret, cfg, false)

		assert.NoError(t, err)

		// Verify deleted env is filtered out
		filteredEnvs := filterEnvsByCurrentEnvironments(
			[]models.Env{
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "staging"}},
				{ProjectEnv: &models.ProjectEnvironmentReference{AlternateID: "deleted-env"}},
			},
			[]models.Environment{
				{AlternateID: "staging"},
			},
		)
		assert.Equal(t, []string{"staging"}, filteredEnvs)
		assert.NotContains(t, filteredEnvs, "deleted-env")
	})
}
