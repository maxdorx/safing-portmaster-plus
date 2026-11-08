# Portmaster Plus

An unofficial, device-local fork of Safing Portmaster, based on **v2.2.8**. Network History and Bandwidth Visibility work without an account. SPN, account controls, subscriptions, news, and promotions are removed from the product interface and disabled in the integrated service.

The existing firewall interface, application rules, DNS filtering, connection monitor, history, and bandwidth charts are retained. Network History still requires enabling its local setting.

Not affiliated with Safing. Portmaster and Safing are upstream trademarks; original copyright, license, and attribution remain. See LICENSE and TRADEMARKS.

## Updates

Application updates are **manual**. Official core/UI updates could overwrite this fork, so the integrated binary updater cannot download or apply them, including from saved settings or the update API. Build and replace the matching fork artifacts together for future releases.

Filter-list and GeoIP/intelligence updates remain automatic. News and promotional notification artifacts are excluded after signed-index verification; protection resources retain upstream verification.

## Download

Get the Windows x64 ZIP from [GitHub Releases](https://github.com/maxdorx/safing-portmaster-plus/releases/latest). It is a replacement bundle for an existing official **Portmaster 2.2.8** installation, not an installer. The ZIP contains `portmaster-core.exe`, `portmaster.exe`, `portmaster.zip`, `assets.zip`, `SHA256SUMS`, and a copy of this README. Replace all four application files together.

## Build on Windows x64

Use Go 1.26 or newer, Node.js/npm, Rust, and the Visual Studio C++ build tools required by upstream Tauri. From this repository run:

```powershell
powershell -ExecutionPolicy Bypass -File packaging/windows/build-plus.ps1
```

The build script produces a matching core, Angular UI archive, native desktop executable, and shared assets under dist. It does not install them or restart the firewall. Existing technical filenames and installation identifiers remain compatible with Portmaster.

## Replace an existing 2.2.8 installation

Use the official **2.2.8** installation and its signed driver. No custom driver, test signing, or Secure Boot changes are needed. Do not replace the driver or native DLL with files from a different version.

1. Download and extract the release ZIP to a local folder. **Do not extract the inner `portmaster.zip` or `assets.zip`.** Close the Portmaster desktop window.
2. Open **PowerShell as administrator** (right-click PowerShell, then choose **Run as administrator**). Change to the extracted release folder. Normal PowerShell cannot stop the service or write to Program Files.
3. Paste the entire block below. It verifies the release checksums, backs up all four current files outside the installation directory, records the startup mode, and temporarily sets the service to Manual. If replacement fails, it restores the backed-up files. The startup mode stays Manual until you verify the result.

```powershell
& {
    $ErrorActionPreference = 'Stop'
    $release = (Get-Location).Path
    $install = 'C:\Program Files\Portmaster'
    $files = @('portmaster-core.exe', 'portmaster.exe', 'portmaster.zip', 'assets.zip')
    $expected = @{}
    Get-Content -LiteralPath (Join-Path $release 'SHA256SUMS') | ForEach-Object {
        if ($_ -match '^([a-fA-F0-9]{64})  (.+)$') { $expected[$matches[2]] = $matches[1] }
    }
    foreach ($file in $files) {
        $actual = (Get-FileHash -LiteralPath (Join-Path $release $file) -Algorithm SHA256).Hash
        if (!$expected.ContainsKey($file) -or $actual -ne $expected[$file]) {
            throw "Checksum mismatch or missing manifest entry: $file"
        }
    }
    $backup = Join-Path $env:USERPROFILE ('Portmaster-backup-' + (Get-Date -Format yyyyMMdd-HHmmss))
    New-Item -ItemType Directory -Path $backup | Out-Null
    $startup = (Get-CimInstance Win32_Service -Filter "Name='PortmasterCore'").StartMode
    $startup | Set-Content -LiteralPath (Join-Path $backup 'startup-mode.txt')
    foreach ($file in $files) {
        Copy-Item -LiteralPath (Join-Path $install $file) -Destination $backup
    }
    Set-Service PortmasterCore -StartupType Manual
    Stop-Service PortmasterCore
    try {
        foreach ($file in $files) {
            Copy-Item -LiteralPath (Join-Path $release $file) -Destination (Join-Path $install $file) -Force
        }
        Start-Service PortmasterCore
    } catch {
        $replacementFailure = $_
        Stop-Service PortmasterCore -ErrorAction SilentlyContinue
        foreach ($file in $files) {
            Copy-Item -LiteralPath (Join-Path $backup $file) -Destination (Join-Path $install $file) -Force
        }
        Start-Service PortmasterCore
        throw $replacementFailure
    }
    Write-Host "Backup: $backup"
    Write-Host "Previous startup mode: $startup. Service is temporarily Manual."
}
```

The firewall can interrupt networking while its service is stopped; prepare the files locally first. If an unexpected reboot happens during replacement, Manual startup prevents a broken core from automatically loading again. Keep the official driver, `portmaster-core.dll`, configuration, and databases unchanged.

4. Open Portmaster Plus. Check internet/DNS access, application rules, Network History, bandwidth charts, settings scrolling, and the tray controls. SPN, news, login, and upgrade buttons should be absent. Version information should identify **Portmaster Plus 2.2.8**.
5. After those checks pass, restore the previous service startup mode. For a normal automatic installation:

```powershell
Set-Service PortmasterCore -StartupType Automatic
```

If the recorded previous mode was Manual, leave it Manual. If it was Disabled, investigate why before enabling startup. Preserve any intentional delayed-start configuration when restoring it.

### Roll back

If the service fails to start or networking fails, leave its startup mode Manual and reboot if necessary. In administrator PowerShell, close the UI, stop PortmasterCore, and copy **all four** files from your timestamped backup back into `C:\Program Files\Portmaster`. Start the service and verify networking before restoring its previous startup mode. Use a complete backup from this installation; a lone `.orig` core may belong to a different version.

Installing an official package later replaces this fork's customizations. Future fork application updates are installed with the same four-file procedure. Filter-list and intelligence updates remain automatic.

## Development checks

```powershell
go test -short ./spn/access ./spn/captain ./service/firewall ./service ./service/netquery
go test ./service/updates -run 'TestManualOnly|TestExcluded|TestLocalArtifact|TestValidate' -count=1
node --test desktop/angular/e2e/local-settings.test.cjs
```

The access tests retain the record-lock regression: saving while holding a record lock can deadlock startup. Do not move persistence inside that lock.

Shared SPN code remains in the source because local feature gates and other modules use its types and events. The integrated firewall cannot activate the SPN. This fork does not provide access to Safing's network.

See [validation results and limitations](docs/portmaster-plus-validation.md) and [RAM measurements](docs/portmaster-plus-memory.md).

## License

GPLv3, as upstream. Distributing binaries requires making the corresponding source available. Upstream documentation is preserved in README.upstream.md.
