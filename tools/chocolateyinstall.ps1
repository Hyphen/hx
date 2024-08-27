$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$version = "$env:ChocolateyPackageVersion"
$url = "https://api.hyphen.ai/api/downloads/hyphen-cli/$version?os=windows"

$packageArgs = @{
  packageName    = $env:ChocolateyPackageName
  unzipLocation  = $toolsDir
  url            = $url
  softwareName   = 'hyphen*'
  checksum       = '' # You'll need to calculate this dynamically or provide it
  fileType       = 'EXE'
  fileFullPath   = Join-Path $toolsDir 'hyphen.exe'
  checksumType   = 'sha256'
}

Get-ChocolateyWebFile @packageArgs

# Create an alias for the CLI
$binPath = Join-Path $toolsDir 'hyphen.exe'
Install-BinFile -Name 'hx' -Path $binPath
