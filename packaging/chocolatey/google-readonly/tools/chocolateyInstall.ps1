$ErrorActionPreference = 'Stop'

$version = $env:ChocolateyPackageVersion
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

$checksumAmd64 = 'CHECKSUM_AMD64_PLACEHOLDER'
$checksumArm64 = 'CHECKSUM_ARM64_PLACEHOLDER'

if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    $arch = 'arm64'
    $checksum = $checksumArm64
} elseif ([Environment]::Is64BitOperatingSystem) {
    $arch = 'amd64'
    $checksum = $checksumAmd64
} else {
    throw "32-bit Windows is not supported. gro requires 64-bit Windows."
}

$url = "https://github.com/open-cli-collective/google-cli/releases/download/v${version}/gro_v${version}_windows_${arch}.zip"

Install-ChocolateyZipPackage -PackageName $env:ChocolateyPackageName `
    -Url $url `
    -UnzipLocation $toolsDir `
    -Checksum $checksum `
    -ChecksumType 'sha256'
