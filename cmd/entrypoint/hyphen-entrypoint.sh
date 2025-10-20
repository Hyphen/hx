#!/bin/sh

verify_command() {
    if ! command -v $1 >/dev/null; then
        echo "Missing command '$1'. Please install $1 into the Docker container." 1>&2
        exit 1
    fi
}

verify_command "wget"
verify_command "sed"

verify_environment() {
    if ! printenv $1 >/dev/null; then
        echo "Missing environment variable: $1" 1>&2
        exit 1
    fi
}

verify_environment "HYPHEN_API_KEY"
verify_environment "HYPHEN_APP_ENVIRONMENT"
verify_environment "HYPHEN_APP_ID"
verify_environment "HYPHEN_PROJECT_ID"
verify_environment "HYPHEN_ORGANIZATION_ID"

if [ $# -eq 0 ]; then
    echo "Missing execution command line. Did you forget a CMD in your Dockerfile?" 1>&2
    exit 1
fi

if ! [ -f ".hyphen/hx" ]; then
    mkdir -p .hyphen

    echo ">>> Determining Hyphen CLI latest version..."

    wget -q -O .hyphen/hyphen-cli-version "https://api.hyphen.ai/api/downloads/hyphen-cli/versions?latest=true"
    if [ $? -ne 0 ]; then
        exit 1
    fi

    version=$(sed -n 's/.*"version":"\([^"]*\).*/\1/p' .hyphen/hyphen-cli-version)
    if [ -z "${version}" ]; then
        exit 1
    fi

    rm .hyphen/hyphen-cli-version

    echo ">>> Downloading Hyphen CLI version $version..."

    wget -q -O ./.hyphen/hx "https://api.hyphen.ai/api/downloads/hyphen-cli/${version}?os=linux"
    if [ $? -ne 0 ]; then
        exit 1
    fi

    chmod +x ".hyphen/hx"
fi

echo ">>> Creating ~/.hx ..."

echo "{"                                                      > ~/.hx
echo "  \"hyphen_api_key\": \"${HYPHEN_API_KEY}\"",          >> ~/.hx
echo "  \"organization_id\": \"${HYPHEN_ORGANIZATION_ID}\"", >> ~/.hx
echo "  \"project_id\": \"${HYPHEN_PROJECT_ID}\"",           >> ~/.hx
echo "  \"app_id\": \"${HYPHEN_APP_ID}\""                    >> ~/.hx
echo "}"                                                     >> ~/.hx

echo ">>> Pulling environment variables..."

./.hyphen/hx pull default --force --yes
if [ $? -ne 0 ]; then
    exit 1
fi

./.hyphen/hx pull "${HYPHEN_APP_ENVIRONMENT}" --force --yes
if [ $? -ne 0 ]; then
    exit 1
fi

echo ">>> Running..."

./.hyphen/hx env run "${HYPHEN_APP_ENVIRONMENT}" --yes -- $@
