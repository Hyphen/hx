param (
    [string]$version
)

# Function to get latest version
function Get-LatestVersion {
    param (
        [string]$packageName
    )
    $apiUrl = "https://api.hyphen.ai/api/downloads/${packageName}/versions?latest=true"
    try {
        $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
        $version = $response.data | Where-Object { $_.latest -eq $true } | Select-Object -ExpandProperty version
        if (-not $version) {
            Write-Error "Failed to get latest version"
            exit 1
        }
        return $version
    } catch {
        Write-Error "Failed to get latest version"
        exit 1
    }
}

# Function to create alias
function Create-Alias {
    param (
        [string]$aliasCommand
    )
    $escapedAliasCommand = [Regex]::Escape($aliasCommand)
    $profilePath = $PROFILE
    if (-not (Test-Path -Path $profilePath)) {
        New-Item -ItemType File -Path $profilePath -Force
    }
    if (-not (Get-Content $profilePath | Select-String -Pattern $escapedAliasCommand)) {
        Add-Content -Path $profilePath -Value "`n$aliasCommand"
        Write-Output "Alias added. Please restart your PowerShell session to apply changes."
    } else {
        Write-Output "Alias already exists in $profilePath"
    }
}

# Main installation function
function Install-CLI {
    param (
        [string]$version
    )
    $packageName = "hyphen-cli"
    $os = "windows"  # Hardcoded to windows

    if (-not $version) {
        $version = Get-LatestVersion -packageName $packageName
    }

    $downloadUrl = "https://api.hyphen.ai/api/downloads/${packageName}/${version}?os=${os}"
    $tempDir = New-Item -Type Directory -Path (Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString()))
    $binaryName = "hyphen.exe"
    $tempFilePath = Join-Path $tempDir $binaryName

    Write-Output "Downloading ${packageName} version ${version} for ${os}..."
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFilePath -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Error "Failed to download the binary from $downloadUrl. The specified version may not exist."
        exit 1
    }

    Write-Output "Installing ${binaryName} to C:\Program Files..."
    $installDir = "C:\Program Files"
    Move-Item -Path $tempFilePath -Destination "$installDir\$binaryName" -Force

    Write-Output "${binaryName} has been successfully installed!"
    Write-Output "You can now run '${binaryName}' from anywhere in your terminal."

    # Add the alias
    $aliasCommand = "Set-Alias -Name hx -Value `"$installDir\$binaryName`""
    Create-Alias -aliasCommand $aliasCommand
}

# Run the installation
Install-CLI -version $version
