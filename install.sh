#!/bin/bash

set -e

# Function to detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    
            if [[ "$(uname -m)" == "arm64" ]]; then
                echo "macos-arm"
            else
                echo "macos"
            fi
            ;;
        CYGWIN*|MINGW32*|MSYS*|MINGW*) echo "windows";;
        *)          echo "unsupported";;
    esac
}

# Function to get latest version
get_latest_version() {
    local package_name="$1"
    local api_url="https://api.hyphen.ai/api/downloads/${package_name}/versions?latest=true"
    local version=$(curl -sSf "$api_url" | grep -o '"version":"[^"]*' | cut -d'"' -f4)
    if [ -z "$version" ]; then
        printf "Failed to get latest version\n" >&2
        exit 1
    fi
    echo "$version"
}

# Function to check which shell is being used
detect_shell() {
    case "$SHELL" in
        */zsh) echo "zsh";;
        */bash) echo "bash";;
        */ksh) echo "ksh";;
        */fish) echo "fish";;
        */csh) echo "csh";;
        */tcsh) echo "tcsh";;
        *) echo "unsupported";;
    esac
}

# Function to create alias
create_alias() {
    local alias_command="alias hx='hyphen'"
    local shell=$(detect_shell)
    local config_file
    local alias_added=false

    case "$shell" in
        zsh) config_file="$HOME/.zshrc";;
        bash) 
            if [ -f "$HOME/.bashrc" ]; then
                config_file="$HOME/.bashrc"
            else
                config_file="$HOME/.bash_profile"
            fi
            ;;
        ksh) config_file="$HOME/.kshrc";;
        fish) config_file="$HOME/.config/fish/config.fish";;
        csh|tcsh) config_file="$HOME/.cshrc";;
        *) 
            printf "\nUnsupported shell. Please add the following alias manually to your shell configuration file:\n"
            printf "  $alias_command\n\n"
            exit 1
            ;;
    esac

    if [ -f "$config_file" ]; then
        if ! grep -q "$alias_command" "$config_file"; then
            printf "\nAdding alias 'hx' for hyphen to $config_file\n"
            echo "$alias_command" >> "$config_file"
            alias_added=true
        else
            printf "\nAlias 'hx' already exists in $config_file\n\n"
            return
        fi
    fi

    if [ "$alias_added" = true ]; then
        printf "\nAlias added. Please source the configuration file to apply changes:\n"
        printf "  source $config_file\n\n"
    else
        printf "\nCould not find a suitable configuration file in your home directory.\n"
        printf "Please add the following alias manually to your shell configuration file:\n"
        printf "  $alias_command\n\n"
    fi
}

# Main installation function
install_cli() {
    local package_name="hyphen-cli"
    local version="$1"
    local os=$(detect_os)
    if [ "$os" = "unsupported" ]; then
        printf "\nUnsupported operating system\n\n"
        exit 1
    fi

    if [ -z "$version" ]; then
        version=$(get_latest_version "$package_name")
    fi

    local download_url="https://api.hyphen.ai/api/downloads/${package_name}/${version}?os=${os}"
    local temp_dir=$(mktemp -d)
    local binary_name="hyphen"

    printf "\nDownloading ${package_name} version ${version} for ${os}...\n"
    curl -sSfL -L "$download_url" -o "${temp_dir}/${binary_name}"
    
    chmod +x "${temp_dir}/${binary_name}"

    # Determine install location based on OS
    local install_dir
    case "$os" in
        linux|macos|macos-arm)
            install_dir="/usr/local/bin"
            ;;
        windows)
            printf "\nWindows installation is not supported by this script.\n\n"
            exit 1
            ;;
    esac

    printf "\nInstalling ${binary_name} to ${install_dir}...\n"
    sudo mv "${temp_dir}/${binary_name}" "${install_dir}/"

    printf "\n${binary_name} has been successfully installed!\n"
    printf "You can now run '${binary_name}' from anywhere in your terminal.\n\n"

    # Add the alias
    create_alias
}

# Run the installation with an optional version parameter
install_cli "$1"
