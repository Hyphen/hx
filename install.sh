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
        echo "Failed to get latest version" >&2
        exit 1
    fi
    echo "$version"
}

# Function to create alias
create_alias() {
    local alias_added=false
    local alias_command="alias hx='hyphen'"

    # Function to add alias to a specific file
    add_alias_to_file() {
        local file="$1"
        if [ -f "$file" ]; then
            echo "Adding alias 'hx' for hyphen to $file"
            echo "$alias_command" >> "$file"
            alias_added=true
        fi
    }

    # Add to .zshrc if it exists
    add_alias_to_file "$HOME/.zshrc"

    # Add to .bashrc if it exists
    add_alias_to_file "$HOME/.bashrc"

    # If neither .zshrc nor .bashrc exist, try .bash_profile
    if [ "$alias_added" = false ]; then
        add_alias_to_file "$HOME/.bash_profile"
    fi

    if [ "$alias_added" = true ]; then
        echo "Alias added. Please restart your terminal or source the relevant configuration file(s) to apply changes."
    else
        echo "Could not find .zshrc, .bashrc, or .bash_profile in your home directory."
        echo "Please add the following alias manually to your shell configuration file:"
        echo "  $alias_command"
    fi
}

# Main installation function
install_cli() {
    local package_name="hyphen-cli"
    local os=$(detect_os)
    if [ "$os" = "unsupported" ]; then
        echo "Unsupported operating system"
        exit 1
    fi

    local version=$(get_latest_version "$package_name")
    local download_url="https://api.hyphen.ai/api/downloads/${package_name}/${version}?os=${os}"
    local temp_dir=$(mktemp -d)
    local binary_name="hyphen"

    echo "Downloading ${package_name} version ${version} for ${os}..."
    curl -sSfL -L "$download_url" -o "${temp_dir}/${binary_name}"
    
    chmod +x "${temp_dir}/${binary_name}"

    # Determine install location based on OS
    local install_dir
    case "$os" in
        linux|macos|macos-arm)
            install_dir="/usr/local/bin"
            ;;
        windows)
            echo "Windows installation is not supported by this script."
            exit 1
            ;;
    esac

    echo "Installing ${binary_name} to ${install_dir}..."
    sudo mv "${temp_dir}/${binary_name}" "${install_dir}/"

    echo "${binary_name} has been successfully installed!"
    echo "You can now run '${binary_name}' from anywhere in your terminal."

    # Add the alias
    create_alias
}

# Run the installation
install_cli
