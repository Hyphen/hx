package env

const DEFAULT_ENV_CONTENTS = `
# This is a placeholder .env file.
# https://docs.hyphen.ai/docs/env-secrets-management

# Variables defined here will be securely managed by Hyphen and can be automatically
# injected into your Docker containers using our entrypoint script.
# See: https://github.com/hyphen/hx/blob/main/cmd/entrypoint/hyphen-entrypoint.sh

# KEY=value
`
