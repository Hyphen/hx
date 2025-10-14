#!/bin/sh

if ! command -v "wget" >/dev/null; then
    echo "Missing command 'wget'. Please install wget into the Docker container." 1>&2
    exit 1
fi

if ! command -v "sed" >/dev/null; then
    echo "Missing command 'sed'. Please install sed into the Docker container." 1>&2
    exit 1
fi

if [ $# -eq 0 ]; then
    echo "Missing execution command line. Did you forget a CMD in your Dockerfile?" 1>&2
    exit 1
fi

if [ -z "${HYPHEN_API_KEY}" ]; then
    echo "Missing environment variable: HYPHEN_API_KEY" 1>&2
    exit 1
fi

if [ -z "${HYPHEN_APP_ENVIRONMENT}" ]; then
    echo "Missing environment variable: HYPHEN_APP_ENVIRONMENT" 1>&2
    exit 1
fi

if ! [ -f ".hyphen/hx" ]; then
    echo ">>> Determining Hyphen CLI latest version..."

    wget -q -O /tmp/hyphen-cli-version "https://api.hyphen.ai/api/downloads/hyphen-cli/versions?latest=true"
    if [ $? -ne 0 ]; then
        exit 1
    fi

    version=$(sed -n 's/.*"version":"\([^"]*\).*/\1/p' /tmp/hyphen-cli-version)
    if [ -z "${version}" ]; then
        exit 1
    fi

    echo ">>> Downloading Hyphen CLI version $version..."

    mkdir -p .hyphen

    wget -q -O ./.hyphen/hx "https://api.hyphen.ai/api/downloads/hyphen-cli/${version}?os=linux"
    if [ $? -ne 0 ]; then
        exit 1
    fi

    chmod +x ".hyphen/hx"
fi

echo ">>> Logging into Hyphen..."

./.hyphen/hx auth --use-api-key
if [ $? -ne 0 ]; then
    exit 1
fi

echo ">>> Pulling environment variables..."

./.hyphen/hx pull "${HYPHEN_APP_ENVIRONMENT}" --force --yes
if [ $? -ne 0 ]; then
    exit 1
fi

echo ">>> Running..."

./.hyphen/hx env run "${HYPHEN_APP_ENVIRONMENT}" --yes -- $@
