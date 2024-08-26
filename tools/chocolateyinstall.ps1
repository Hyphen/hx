$ErrorActionPreference = 'Stop'
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url        = 'https://api.hyphen.ai/api/downloads/hyphen-cli/$version$?os=windows'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url           = $url
  softwareName  = 'hyphen*'
  checksum      = ''
  checksumType  = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Create an alias for the CLI
$binPath = Join-Path $toolsDir 'hyphen.exe'
Install-BinFile -Name 'hx' -Path $binPath

