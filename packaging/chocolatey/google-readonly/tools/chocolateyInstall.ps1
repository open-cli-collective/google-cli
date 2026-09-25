$ErrorActionPreference = 'Stop'

$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

$urlAmd64 = 'URL_AMD64_PLACEHOLDER'
$urlArm64 = 'URL_ARM64_PLACEHOLDER'
$checksumAmd64 = 'CHECKSUM_AMD64_PLACEHOLDER'
$checksumArm64 = 'CHECKSUM_ARM64_PLACEHOLDER'

if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    $url = $urlArm64
    $checksum = $checksumArm64
} elseif ([Environment]::Is64BitOperatingSystem) {
    $url = $urlAmd64
    $checksum = $checksumAmd64
} else {
    throw "32-bit Windows is not supported. gro requires 64-bit Windows."
}

Install-ChocolateyZipPackage -PackageName $env:ChocolateyPackageName `
    -Url $url `
    -UnzipLocation $toolsDir `
    -Checksum $checksum `
    -ChecksumType 'sha256'
