# Hyphen CLI Command Reference


## Env variables
- `HYPHEN_CUSTOM_AUTH`: this should be the dev base url, example: `https://dev-auth.hyphen.ai`
- `HYPHEN_CUSTOM_APIX`: this should be the dev base url, example: `https://dev-api.hyphen.ai`

## Main Commands

### `hyphen`

The root command for the Hyphen CLI.

Usage:
```bash
hyphen [command]
```

Global Flags:
  - --org: Organization ID (default is used if not provided)
  - --yes, -y: Automatically answer yes for prompts

Available Commands:
  - `auth`: Authenticate with Hyphen
  - `config`: Manage Hyphen CLI configuration
  - `init`: Initialize a project
  - `members`: Manage organization members
  - `organization`: Manage organizations
  - `project`: Manage projects
  - `update`: Update the Hyphen CLI
  - `version`: Display the version of the Hyphen CLI

## Authentication Command
`hyphen auth`
Authenticate with Hyphen.
Usage:

```bash
hyphen auth
```
This command starts the OAuth flow and saves the credentials.
Configuration Commands


## Configuration Commands
`hyphen config`
Manage Hyphen CLI configuration.
Usage:
```bash
hyphen config [command]
```
Available Subcommands:
  - `set`: Set configuration values

`hyphen config set organization-id <id>`
Set the organization ID for the Hyphen CLI.
Usage:
```bash
hyphen config set organization-id <id>
```


## Initialization Command
`hyphen init`
Initialize a project.
Usage:
```bash
hyphen init
```
This command creates a new project and initializes the manifest file.

## Member Management Commands
`hyphen members`
Manage organization members.
Usage:
```bash
hyphen members [command]
```

Available Subcommands:
  - `create`: Create a new member
  - `delete`: Delete a member
  - `list`: List all members


`hyphen members create`

Create a new member in the organization.
Usage:
```bash
hyphen members create --firstName <firstName> --lastName <lastName> --email <email>
```
Flags:
  - `--firstName, -f`: First name of the new member (required)
  - `--lastName, -l`: Last name of the new member (required)
	- `--email, -e`: Email of the new member (required)

`hyphen members delete <member-id>`

Delete a member from the organization.
Usage:
```bash
hyphen members delete <member-id>
```
Aliases: `del`


`hyphen members list`
List all members of an organization.
Usage:
```bash
hyphen members list [flags]
```
Aliases: `ls`
Flags:
  - `--pageNum, -n`: Page number (default 1)
  - `--pageSize, -s`: Page size (default 10)


## Organization Management Commands
`hyphen organization`
Manage organizations.
Usage:
```bash
hyphen organization [command]
```
Available Subcommands:
  - `list`: List all organizations

`hyphen organization list`
List all organizations.
Usage:
```bash
hyphen organization list
```
Aliases: `ls`


## Project Management Commands
`hyphen project`
Manage projects.
Usage:
```bash
hyphen project [command]
```
Available Subcommands:
  - `list`: List all projects

`hyphen project list`
List all projects.
Usage:
```bash
hyphen project list [flags]
```
Aliases: `ls`
Flags:
  - `--pageNum`, -n: Page number (default 1)
  - `--pageSize`, -s: Page size (default 10)

## Other Commands

`hyphen update`
Update the Hyphen CLI.
Usage:
```bash
hyphen update
```
`hyphen version`
Display the version of the Hyphen CLI.
Usage:
```bash
hyphen version
```
