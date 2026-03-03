# Hyphen CLI Command Reference

## Env variables
- `HYPHEN_DEV`: set to `true` if you wish to interact against the Hyphen dev environment. You can also use `--dev`, but it would be required with each command.

## Installation
**Linux/MacOS**
```bash
sh -c "$(curl -fsSL https://cdn.hyphen.ai/install/install.sh)"
```

**Windows**
```powershell
powershell -c "irm https://cdn.hyphen.ai/install/install.ps1 | iex"
```

## Main Commands

### `hyphen`
The root command for the Hyphen CLI.

Usage:
```bash
hyphen [command]
```

Global Flags:
-   `--org`: Organization ID (e.g., org_123)
-   `--proj`: Project ID (e.g., proj_123)
-   `--env`: Environment ID (e.g., env_12345)
-   `--yes, -y`: Automatically answer yes for prompts
-   `--no`: Automatically answer no for prompts

Available Commands:
-   `auth`: Authenticate with Hyphen
-   `init`: Initialize an app
-   `update`: Update the Hyphen CLI
-   `config`: Manage CLI settings
-   `set-org`: Set the organization ID
-   `set-project`: Set the project ID
-   `version`: Display the version of the Hyphen CLI
-   `push`: Upload and encrypt environment variables for a specific environment
-   `pull`: Retrieve and decrypt environment variables for a specific environment
-   `link`: Shorten a URL and optionally generate a QR code
-   `app`: Manage applications
-   `project`: Manage projects
-   `env`: Manage environments

## Authentication Command
### `hyphen auth`
Authenticate with Hyphen.

Usage:
```bash
hyphen auth
```
This command starts the OAuth flow and saves the credentials.

### API Key authentication
If you are authenticating in a CI/CD environment and need to authenticate using an API key, you can do so in 2 ways:

#### `--use-api-key`
This will look for the `HYPHEN_API_KEY` environment variable first and if not found it will prompt for the key.

Usage:
```bash
hyphen auth --use-api-key
```

#### `--set-api-key VALUE`
This will expect an inline value. The risk in using this method is just in exposing the secret in your terminal output, but it's provided for convenience.

Usage:
```bash
hyphen auth --set-api-key VALUE
```

## Initialization Command
### `hyphen init`
Initialize an app.

Usage:
```bash
hyphen init <app name>
```
This command creates a new app and initializes the manifest file.

## Push Command
### `hyphen push`
Upload and encrypt environment variables for a specific environment.

Usage:
```bash
hyphen push [flags]
```

Flags:
-   `--environment, -e string`: Specify the environment to push to (e.g., dev, staging, prod)
-   `--org string`: Specify the organization ID (overrides the default from credentials)

This command reads the local .env file corresponding to the specified environment, encrypts the variables, and uploads them to the Hyphen platform.

## Pull Command
### `hyphen pull`
Retrieve and decrypt environment variables for a specific environment.

Usage:
```bash
hyphen pull [flags]
```

Flags:
-   `--environment, -e string`: Specify the environment to pull from (e.g., dev , staging, prod)
-   `--org string`: Specify the organization ID (overrides the default from credentials)

This command retrieves the encrypted environment variables from the specified environment, decrypts them, and saves them to a local .env file.

## Update Command
### `hyphen update`
Update the Hyphen CLI.

Usage:
```bash
hyphen update
```

The CLI can also auto-update on regular command execution. Auto-update is skipped in CI environments and can be turned off with:

```bash
hyphen config auto-update off
```

## Config Command
### `hyphen config auto-update`
Enable or disable automatic updates for the CLI by writing to global `.hx` config.

Usage:
```bash
hyphen config auto-update <on|off>
```

## Set Organization Command
### `hyphen set-org`
Set the organization ID in .hx.

Usage:
```bash
hyphen set-org <id>
```

## Set Project Command
### `hyphen set-project`
Set the project ID in .hx.

Usage:
```bash
hyphen set-project <id>
```

## Version Command
### `hyphen version`
Display the version of the Hyphen CLI.

Usage:
```bash
hyphen version
```

## Link Command
### `hyphen link`
Shorten a URL and optionally generate a QR code.

Usage:
```bash
hyphen link <long_url> [flags]
```

Flags:
-   `--qr`: Generate a QR code for the shortened URL
-   `--domain string`: Specify a custom domain for the short URL (default: organization's default domain)
-   `--tag strings`: Add tags to the shortened URL (can be used multiple times)
-   `--code string`: Set a custom short code for the URL (if available)
-   `--title string`: Add a title to the shortened URL

## App Command
### `hyphen app`
Manage applications.

Available Subcommands:
-   `list`: List all applications associated with the organization and project
-   `create`: Create a new app
-   `get`: Get an app

#### List Command
### `hyphen app list`
List all applications associated with the organization and project.

Usage:
```bash
hyphen app list
```

#### Create Command
### `hyphen app create`
Create a new app within your organization.

Usage:
```bash
hyphen app create <app name> [flags]
```

Flags:
-   `--id, -i`: Specify a custom app ID (optional)

Examples:
```bash
hyphen app create myapp
hyphen app create myapp --id custom-app-id
```

#### Get Command
### `hyphen app get`
Retrieve details of an app within your organization.

Usage:
```bash
hyphen app get <id>
```

Examples:
```bash
hyphen app get custom-app-id
```

## Project Command
### `hyphen project`
Manage projects.

Available Subcommands:
-   `list`: List all projects
-   `create`: Create a new project with the provided name
-   `get`: Get a project by ID

#### List Command
### `hyphen project list`
List all projects.

Usage:
```bash
hyphen project list
```

#### Create Command
### `hyphen project create`
Create a new project with the provided name.

Usage:
```bash
hyphen project create [name]
```

Examples:
```bash
hyphen project create "project"
```

#### Get Command
### `hyphen project get`
Get a project by ID.

Usage:
```bash
hyphen project get [project_id]
```

Examples:
```bash
hyphen project get proj_123
```

## Env Command
### `hyphen env`
Manage environment .env secrets.

Available Subcommands:
-   `pull`: Retrieve and decrypt .env secrets for a specific environment
-   `push`: Upload and encrypt .env secrets for a specific environment

#### Pull Command
### `hyphen env pull`
Retrieve and decrypt .env secrets for a specific environment.

Usage:
```bash
hyphen env pull [flags]
```

Flags:
-   `--environment, -e string`: Specify the environment to pull from (e.g., dev, staging, prod)
-   `--org string`: Specify the organization ID (overrides the default from credentials)
-   `--all`: Pull secrets for all environments

This command retrieves the encrypted environment variables from the specified environment, decrypts them, and saves them to a local .env file.

#### Push Command
### `hyphen env push`
Upload and encrypt .env secrets for a specific environment.

Usage:
```bash
hyphen env push [flags]
```

Flags:
-   `--environment, -e string`: Specify the environment to push to (e.g., dev, staging, prod)
-   `--org string`: Specify the organization ID (overrides the default from credentials)
-   `--all`: Push secrets for all environments

This command reads the local .env file corresponding to the specified environment, encrypts the variables, and uploads them to the Hyphen platform.

#### Run Command
### `hyphen env run production -- yourcommand and command args`

Run a sub-command with the specified environment set as environment variables, merging with defaults.

Usage:
```bash
hyphen env run [environment] -- [command]
```

`environment` is optional. Skipping it will just use the default (`.env`) environment.

When they exist, the files will be loaded and appended to the environment in this order:
- `.env`
- `.env.local`
- `.env.{environment}`

So a variable defined in `.env` that is also defined in `.env.{environment}` will have the `.env.{environment}` value take precedence.
