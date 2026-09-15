# Hyphen CLI Command Reference

## Env variables
- `HYPHEN_DEV`: set to `true` if you wish to interact against the Hyphen dev environment. You can also use `--dev`, but it would be required with each command.
- `HYPHEN_LOCAL_APIX`: set to `true` for local APIX (`http://localhost:4000`), Socket.IO (`http://localhost:4100`), and app links (`http://localhost:3000`). This uses **dev authentication** (`https://dev-auth.hyphen.ai`) and the dev OAuth client without changing the project Horizon endpoint.
- `HYPHEN_LOCAL_HORIZON`: set to `true` for your project's local Horizon requests (`http://localhost:3333`), including environment/dot-env reads, without changing APIX or authentication.
- `HYPHEN_LOCAL`: set to `true` to force both local switches on, even if either individual switch is `false`. Local endpoints take precedence over `HYPHEN_DEV`/`--dev`. Variable names are uppercase and case-sensitive.

The CLI's own feature-flag provider **always uses production Horizon** through
the SDK's organization-specific production endpoint. None of these local
switches change that endpoint. `HYPHEN_DEV` still selects its `development`
evaluation environment; otherwise it evaluates `production`.

After rebuilding hx, authenticate again to replace credentials obtained from production:

```sh
export HYPHEN_LOCAL=true
hx auth
```

For **local APIX with dev project Horizon and dev authentication**:

```sh
export HYPHEN_LOCAL=false
export HYPHEN_LOCAL_APIX=true
export HYPHEN_LOCAL_HORIZON=false
export HYPHEN_DEV=true
hx auth
```

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

## Build and deploy

`hx` and `hyphen` support the same commands. Run these from your app's `.hx`
directory. `--type` selects the **local build**: `docker` (default) or `static`.
Docker keeps the existing Dockerfile generation, build, registry push, and build
registration flow. Use `--dockerfile/-f` to choose a Dockerfile.

```sh
hx build
hx deploy
hx build --type static ./dist
hx deploy --type static ./dist
hx deploy dply_123 --type static ./dist
```

For static builds, the final positional argument is a **required directory**
containing the already-built website. hx does not run a frontend build tool or
Docker. Build your frontend first. Supply a directory, not an archive, symlink,
or individual `.gz` file.
An empty directory, symlinks within it, non-regular files, or paths containing
`\\`, `%`, `?`, `#`, control characters, or invalid UTF-8 fail validation.

The project must have exactly one ready HyphenCloud **SiteRegistry** connection.
hx hashes the original files and stages a gzip copy of each in a temporary
directory, then requests a short-lived upload capability through APIX. It sends
each gzip file as multipart directly to the returned upload URL, finalizes the
revision, validates the sealed receipt, and registers a `Static` build artifact.
File bytes do not pass through APIX or nFabric. The upload uses only its
capability bearer; APIX credentials are not sent to the upload host, and upload
redirects are refused. Temporary gzip files are removed on completion or failure.

The returned registry limits apply to file count, decompressed file sizes, and
multipart request size. Failed uploads do not register a build. hx attempts to
abort unfinished uploads; an expired/rejected capability requires rerunning the
build. Once sealed, the revision is not aborted, including if build registration
fails. No deployment runs until build registration succeeds.

### Selecting apps and sites

`--apps` selects **container apps**; `--sites` selects **static sites**. Both use
comma-separated app IDs or alternate IDs, with optional build selectors:
`latest`, `lastDeployed`, `latestPreview`, or an `abld_...` build ID.

```sh
hx deploy dply_123 --no-build --apps api:latest --sites website:abld_123
hx deploy dply_123 --no-build --sites website:lastDeployed
hx deploy dply_123 --type static --apps api:latest --sites website ./dist
```

An unqualified selector matching the local `.hx` app builds that app when its
kind matches `--type`. Other unqualified selectors use `latest`. Explicit build
selectors always use that build, including with `--no-build`. Static local
builds require the local site to be selected without a build selector; otherwise
use `--no-build`. No directory is needed or used with `--no-build`.

When either selection flag is supplied, only those selected apps/sites are
submitted to the run. When neither is supplied, all deployment apps and sites
are submitted: the local app uses its new build unless `--no-build`, and other
members use `latest`. Static auto-add/create writes the app under `sites` with
the HyphenCloud **organization integration ID** in `deploymentSettings.targets`;
the registry connection ID is only used for artifacts/uploads. Existing app and
site settings are preserved. An existing container app cannot silently be moved
to `sites`, or vice versa; update deployment settings explicitly first.

Without a deployment ID, hx resolves the project's development environment (or
`--env`) and creates its deployment if needed. Preview deployment keeps the
existing `--preview/-r` and `--prefix/-x` flow. The selected preview's host prefix
is sent as the upload `previewHash`, together with the environment ID. Standalone
`hx build --type static --env env_123 --preview my-branch ./dist` associates the
build with the environment/preview name without creating or activating a preview.

Both commands retain `--output json` for machine-readable results and errors.
Build/deploy commands remain subject to the existing deployments feature toggle.

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
