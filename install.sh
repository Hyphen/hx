#!/bin/sh

set -e

# Function to detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    
            if [ "$(uname -m)" = "arm64" ]; then
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
    package_name="$1"
    api_url="https://api.hyphen.ai/api/downloads/${package_name}/versions?latest=true"
    version=$(curl -sSf "$api_url" | sed -n 's/.*"version":"\([^"]*\).*/\1/p')
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

# Function to create alias and add ~/.local/bin to PATH
update_shell_config() {
    alias_command="alias hx='hyphen'"
    path_update='export PATH="$HOME/.local/bin:$PATH"'
    shell=$(detect_shell)
    config_file=""
    changes_made=false

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
        fish) 
            config_file="$HOME/.config/fish/config.fish"
            path_update='set -gx PATH "$HOME/.local/bin" $PATH'
            ;;
        csh|tcsh) config_file="$HOME/.cshrc";;
        *) 
            printf "\nUnsupported shell. Please add the following manually to your shell configuration file:\n"
            printf "  %s\n" "$alias_command"
            printf "  %s\n\n" "$path_update"
            return
            ;;
    esac

    if [ -f "$config_file" ]; then
        if ! grep -q "$alias_command" "$config_file"; then
            printf "\nAdding alias 'hx' for hyphen to %s\n" "$config_file"
            echo "$alias_command" >> "$config_file"
            changes_made=true
        fi
        if ! grep -q "$path_update" "$config_file"; then
            printf "Adding ~/.local/bin to PATH in %s\n" "$config_file"
            echo "$path_update" >> "$config_file"
            changes_made=true
        fi
    fi

    if [ "$changes_made" = true ]; then
        printf "\nChanges made to %s. Please run the following command to apply changes:\n" "$config_file"
        printf "  source %s\n\n" "$config_file"
    else
        printf "\nNo changes were necessary in %s\n\n" "$config_file"
    fi
}

# Main installation function
install_cli() {
    package_name="hyphen-cli"
    version="$1"
    os=$(detect_os)
    if [ "$os" = "unsupported" ]; then
        printf "\nUnsupported operating system\n\n"
        exit 1
    fi

    if [ -z "$version" ]; then
        version=$(get_latest_version "$package_name")
    fi

    download_url="https://api.hyphen.ai/api/downloads/${package_name}/${version}?os=${os}"
    temp_dir=$(mktemp -d)
    binary_name="hyphen"

    printf "\nDownloading %s version %s for %s...\n" "$package_name" "$version" "$os"
    curl -sSfL -L "$download_url" -o "${temp_dir}/${binary_name}"
    
    chmod +x "${temp_dir}/${binary_name}"

    # Install to ~/.local/bin
    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"

    printf "\nInstalling %s to %s...\n" "$binary_name" "$install_dir"
    mv "${temp_dir}/${binary_name}" "${install_dir}/"

    printf "\n%s has been successfully installed!\n" "$binary_name"
    printf "You can now run '%s' from anywhere in your terminal once you've updated your PATH.\n\n" "$binary_name"

    # Update shell configuration
    update_shell_config
}

# Run the installation with an optional version parameter
install_cli "$1"
