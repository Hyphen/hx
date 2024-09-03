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
  - `--yes, -y`: Automatically answer yes for prompts

Available Commands:
  - `auth`: Authenticate with Hyphen
  - `init`: Initialize an app
  - `update`: Update the Hyphen CLI
  - `use-org`: Set the organization ID
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


## Update Command
`hyphen update`
Update the Hyphen CLI.
Usage:
```bash
hyphen update
```

## Use Organization Command
`hyphen use-org`
Set the organization ID.
Usage:
```bash
hyphen use-org <id>
```

## Version Command
`hyphen version`
Display the version of the Hyphen CLI.
Usage:
```bash
hyphen version
```
