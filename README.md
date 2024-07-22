# Hyphen CLI Command Reference

## Main Commands

## Env variables
- `HYPHEN_CUSTOM_AUTH`: this should be the dev base url, example: `https://dev-auth.hyphen.ai`
- `HYPHEN_CUSTOM_ENV`: this should be the dev base url, example: `https://dev-api.hyphen.ai/env`


### `hyphen`

The root command for the Hyphen CLI.

Usage:
```bash
hyphen [command]
```

Available Commands:
- `version`: Display the version of the Hyphen CLI
- `update`: Update the Hyphen CLI
- `env`: Environment-related commands
- `init`: Initialize the Hyphen CLI

## Environment Commands

### `hyphen env`

Manage environments and environment variables.

Usage:
```bash
hyphen env [command]
```


Available Commands:
- `init`: Initialize the environment
- `create`: Create a new environment file
- `decrypt`: Decrypt environment variables
- `encrypt`: Encrypt environment variables
- `merge`: Merge environment variables
- `pull`: Pull environment variables
- `push`: Push environment variables
- `run`: Run a command with environment variables

### `hyphen env init`

Initialize the environment with necessary configurations.

Usage:
```bash
hyphen env init
```


### `hyphen env create [ENVIRONMENT]`

Create a new environment file for the specified environment.

Usage:
```bash
hyphen env create [ENVIRONMENT]
```
Example:
```bash
hyphen env create production
```
or 
```bash
#This will create de deafult env: is like typing hyphen env default 
hyphen env create 
```

### `hyphen env decrypt`

Decrypt environment variables.

Usage:
```bash
hyphen env decrypt -s [STRING]
hyphen env decrypt -f [FILE]
```
Flags:
- `-s, --string`: String to decrypt
- `-f, --file`: File to decrypt

### `hyphen env encrypt [FILE]`

Encrypt a file containing environment variables.

Usage:
```bash
hyphen env encrypt [FILE]
```


### `hyphen env merge [ENVIRONMENT] [FILE]`

Merge environment variables into a file.

Usage:
```bash
hyphen env merge [ENVIRONMENT] [FILE]
```

Flags:
- `-f, --file`: Specify the output file name

### `hyphen env push [ENVIRONMENT]`

Push an existing environmental variable file to Hyphen.

Usage:
```bash
hyphen env push [ENVIRONMENT]
```
Flags:
- `-f, --file`: Specify the file to push

### `hyphen env run [ENVIRONMENT] [COMMAND] [ARGS...]`

Run a command using environment variables from a specified environment.

Usage:
```bash
hyphen env run [ENVIRONMENT] [COMMAND] [ARGS...]
```
Flags:
- `-f, --file`: Specific environment file to use
- `-s, --stream`: Stream environment variables

## Initialization Command

### `hyphen init`

Initialize the Hyphen CLI.

Usage:
```bash
hyphen init
```

This command sets up environment variables and aliases for the CLI tool.

## Other Commands

### `hyphen version`

Display the version of the Hyphen CLI.

Usage:
```bash
hyphen version
```
