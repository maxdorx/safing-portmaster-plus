<#
.SYNOPSIS
Builds Portmaster Plus core, Angular archive, and desktop executable without installing them.
.EXAMPLE
powershell -ExecutionPolicy Bypass -File packaging/windows/build-plus.ps1
.EXAMPLE
powershell -ExecutionPolicy Bypass -File packaging/windows/build-plus.ps1 -Stage Desktop
.NOTES
Requires Go, Node/npm, Rust (MSVC), and Visual Studio C++ Build Tools + Windows SDK.
The desktop stage requires a previously built Angular tauri-builtin distribution.
No service, driver, installed application, or firewall configuration is modified.
#>
[CmdletBinding()]
param(
    [ValidateSet('All', 'Core', 'UI', 'Desktop')]
    [string]$Stage = 'All',
    [switch]$SkipNpmCi,
    [string]$OutputDirectory
)

$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
if (!$OutputDirectory) {
    $OutputDirectory = Join-Path $projectRoot 'dist/portmaster-plus/windows_amd64'
}
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
$angularDirectory = Join-Path $projectRoot 'desktop/angular'
$tauriDirectory = Join-Path $projectRoot 'desktop/tauri/src-tauri'
$version = (Get-Content (Join-Path $angularDirectory 'package.json') -Raw | ConvertFrom-Json).version
$cargoManifest = Get-Content (Join-Path $tauriDirectory 'Cargo.toml') -Raw
if ($cargoManifest -notmatch ('(?m)^version = "' + [regex]::Escape($version) + '"')) {
    throw 'Angular and desktop versions must match before building.'
}

function Invoke-BuildCommand {
    param([string]$Program, [string[]]$Arguments)
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Program failed with exit code $LASTEXITCODE"
    }
}

function New-PortableArchive {
    param([string]$SourceDirectory, [string]$Destination)
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    Add-Type -AssemblyName System.IO.Compression
    $sourceRoot = [IO.Path]::GetFullPath($SourceDirectory).TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
    if (Test-Path -LiteralPath $Destination) { Remove-Item -LiteralPath $Destination -Force }
    $archive = [IO.Compression.ZipFile]::Open($Destination, [IO.Compression.ZipArchiveMode]::Create)
    try {
        Get-ChildItem -LiteralPath $sourceRoot -File -Recurse -Force | ForEach-Object {
            # Windows PowerShell's CreateFromDirectory uses backslashes in ZIP names.
            # Core asset lookups require the portable forward-slash ZIP convention.
            $entryName = $_.FullName.Substring($sourceRoot.Length).Replace('\', '/')
            [IO.Compression.ZipFileExtensions]::CreateEntryFromFile($archive, $_.FullName, $entryName, [IO.Compression.CompressionLevel]::Optimal) | Out-Null
        }
    }
    finally { $archive.Dispose() }
}

New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
$originalDirectory = Get-Location
$savedEnvironment = @{}
foreach ($name in @('CGO_ENABLED', 'GOOS', 'GOARCH', 'CARGO_HOME', 'CARGO_TARGET_DIR')) {
    $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
try {
    Set-Location $projectRoot
    if ($Stage -in @('All', 'Core')) {
        $env:CGO_ENABLED = '0'
        $env:GOOS = 'windows'
        $env:GOARCH = 'amd64'
        $buildTime = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
        $flags = "-X github.com/safing/portmaster/base/info.version=$version -X github.com/safing/portmaster/base/info.buildSource=portmaster-plus -X github.com/safing/portmaster/base/info.buildTime=$buildTime"
        Invoke-BuildCommand 'go' @('build', '-trimpath', '-ldflags', $flags, '-o', (Join-Path $OutputDirectory 'portmaster-core.exe'), './cmds/portmaster-core')
    }

    if ($Stage -in @('All', 'UI')) {
        Set-Location $angularDirectory
        if (!$SkipNpmCi) { Invoke-BuildCommand 'npm.cmd' @('ci', '--no-audit', '--no-fund') }
        Invoke-BuildCommand 'npm.cmd' @('run', 'build-libs')
        # Archive only the main UI; the built-in UI is embedded in portmaster.exe.
        $mainDistribution = Join-Path $projectRoot 'dist/build-cache/angular-main'
        Invoke-BuildCommand 'npm.cmd' @('exec', '--', 'ng', 'build', '--configuration', 'production', '--base-href', '/ui/modules/portmaster/', '--output-path', $mainDistribution, 'portmaster')
        $archive = Join-Path $OutputDirectory 'portmaster.zip'
        New-PortableArchive $mainDistribution $archive
        $assetArchive = Join-Path $OutputDirectory 'assets.zip'
        New-PortableArchive (Join-Path $projectRoot 'assets/data') $assetArchive
        Invoke-BuildCommand 'npm.cmd' @('exec', '--', 'ng', 'build', '--configuration', 'production', '--base-href', '/', 'tauri-builtin')
    }

    if ($Stage -in @('All', 'Desktop')) {
        if (!(Test-Path -LiteralPath (Join-Path $angularDirectory 'dist/tauri-builtin/index.html'))) {
            throw 'Build the Angular tauri-builtin distribution first (-Stage UI).'
        }
        Set-Location $tauriDirectory
        if (!$env:CARGO_HOME) { $env:CARGO_HOME = Join-Path $projectRoot 'dist/build-cache/cargo' }
        if (!$env:CARGO_TARGET_DIR) { $env:CARGO_TARGET_DIR = Join-Path $tauriDirectory 'target' }
        # custom-protocol embeds frontendDist, as cargo tauri build --no-bundle does.
        Invoke-BuildCommand 'cargo' @('build', '--release', '--locked', '-j', '2', '--features', 'custom-protocol', '--target', 'x86_64-pc-windows-msvc')
        $desktopExecutable = Join-Path $env:CARGO_TARGET_DIR 'x86_64-pc-windows-msvc/release/portmaster.exe'
        Copy-Item -LiteralPath $desktopExecutable -Destination (Join-Path $OutputDirectory 'portmaster.exe') -Force
    }

    Get-ChildItem -LiteralPath $OutputDirectory -File | Where-Object Name -ne 'SHA256SUMS' | ForEach-Object {
        $hash = Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256
        '{0}  {1}' -f $hash.Hash.ToLowerInvariant(), $_.Name
    } | Set-Content -LiteralPath (Join-Path $OutputDirectory 'SHA256SUMS') -Encoding ASCII
    Write-Host "Portmaster Plus $version build completed: $OutputDirectory"
}
finally {
    Set-Location $originalDirectory
    foreach ($name in $savedEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process')
    }
}
