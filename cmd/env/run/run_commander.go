package run

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Hyphen/cli/internal/environment"
)

type RunCommander struct {
	envHanler environment.EnviromentHandler
}

func InitRunCommander() *RunCommander {
	return &RunCommander{
		envHanler: environment.Restore(),
	}
}

func (r *RunCommander) getEnvironmentVariables(env string) ([]string, error) {
	if StreamVars {
		return r.readEnvFileStremed(env)
	}
	return readEnvFile(env)
}

func (r *RunCommander) readEnvFileStremed(env string) ([]string, error) {
	return r.envHanler.DecryptEnvironmentVars(env)
}

func (r *RunCommander) execute(command string, args []string, envVars []string) error {
	cmd := exec.Command(os.ExpandEnv(command), expandArgs(args)...)
	cmd.Env = append(os.Environ(), envVars...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}
