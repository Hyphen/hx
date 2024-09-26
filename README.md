# Hyphen CLI Command Reference

## Env variables
-   `HYPHEN_CUSTOM_AUTH`: this should be the dev base URL, example: `https://dev-auth.hyphen.ai`
-   `HYPHEN_CUSTOM_APIX`: this should be the dev base URL, example: `https://dev-api.hyphen.ai`

## Main Commands

### `hyphen`

The root command for the Hyphen CLI.

Usage:
```bash
hyphen [command]
```

Global Flags:
  - `--org`: Organization ID (e.g., org_123)
  - `--proj`: Project ID (e.g., proj_123)
  - `--env`: Environment ID (e.g., env_12345)
  - `--api-key`: API Key (e.g, key_123)
  - `--yes, -y`: Automatically answer yes for prompts

Available Commands:
  - `auth`: Authenticate with Hyphen
  - `init`: Initialize an app
  - `update`: Update the Hyphen CLI
  - `set-org`: Set the organization ID
  - `version`: Display the version of the Hyphen CLI

## Authentication Command
`hyphen auth`
Authenticate with Hyphen.
Usage:

```bash
hyphen auth
```
This command starts the OAuth flow and saves the credentials.

## Initialization Command
`hyphen init`
Initialize an app.
Usage:
```bash
hyphen init <app name> 
```
This command creates a new app and initializes the manifest file.


## Push Command
`hyphen push`
Upload and encrypt environment variables for a specific environment.
Usage:
```bash
hyphen push [flags]
```

Flags:
  - `--env string`: Specify the environment to push to (e.g., dev, staging, prod)
  - `--org string`: Specify the organization ID (overrides the default from credentials)

This command reads the local .env file corresponding to the specified environment, encrypts the variables, and uploads them to the Hyphen platform.

## Pull Command
`hyphen pull`
Retrieve and decrypt environment variables for a specific environment.
Usage:
```bash
hyphen pull [flags]
```

Flags:
  - `--env string`: Specify the environment to pull from (e.g., dev, staging, prod)
  - `--org string`: Specify the organization ID (overrides the default from credentials)

This command retrieves the encrypted environment variables from the specified environment, decrypts them, and saves them to a local .env file.


## Update Command
`hyphen update`
Update the Hyphen CLI.
Usage:
```bash
hyphen update
```

## Set Organization Command
`hyphen set-org`
Set the organization ID in .hx.
Usage:
```bash
hyphen set-org <id>
```

## Version Command
`hyphen version`
Display the version of the Hyphen CLI.
Usage:
```bash
hyphen version
```
