# Build Portmaster Plus on Windows

From the repository root, with Go, Node/npm, Rust's MSVC toolchain, and Visual
Studio C++ Build Tools (including a Windows SDK) installed, run:

```powershell
powershell -ExecutionPolicy Bypass -File packaging/windows/build-plus.ps1
```

This builds the core, shared Angular libraries, main UI archive, built-in UI, and
desktop executable at the version declared in the project manifests. It writes
`portmaster-core.exe`, `portmaster.zip`, `assets.zip`, `portmaster.exe`, and `SHA256SUMS` into
`dist/portmaster-plus/windows_amd64`. It downloads dependencies as needed and
does not install or start any service or driver, or modify the live firewall.
The technical executable and archive names remain compatible with Portmaster.

Use `-Stage Core`, `-Stage UI`, or `-Stage Desktop` to rebuild one stage. Desktop
requires the built-in Angular UI from the UI stage. `-SkipNpmCi` reuses installed
Node dependencies. The desktop stage uses Cargo directly, so Tauri CLI is not
required. Native drivers, runtime DLLs, and intelligence data are separate
inputs required for an installable bundle; this script does not produce or sign
an installer.

# Generate Windows installer

## Prerequisites

Earthly release prep step must be executed and the output `dist` folder should be present in the root directory of the repository. (Probably needs to be done on separate machine running linux or downloaded from the CI)
```
  earthly +release-prep
```

## Building the installers

In the root directory of the repository, run the PowerShell script to generate the installers:
```
./packaging\windows\generate_windows_installers.ps1
```

This will output both .exe (NSIS) and .msi (WIX) installers inside the dist folder:
```
...\Portmaster\dist\windows_amd64\Portmaster_0.1.0_x64-setup.exe
...\Portmaster\dist\windows_amd64\Portmaster_0.1.0_x64_en-US.msi
```

## Manual build

### Prerequisites

Ensure you have Rust and Cargo installed.
Install Tauri CLI by running:
```
cargo install tauri-cli --version "^2.0.0" --locked
```

### Folder structure

Create binary and intel folder inside the tauri project folder and place all the necessary files inside.
The folder structure should look like this:
```
...\Portmaster\desktop\tauri\src-tauri\binary
    assets.zip
    index.json
    portmaster-core.dll
    portmaster-core.exe
    portmaster-kext.dll
    portmaster-kext.sys
    portmaster.zip
    WebView2Loader.dll

...\Portmaster\desktop\tauri\src-tauri\intel
    base.dsdl
    geoipv4.mmdb
    geoipv6.mmdb
    index.dsd
    index.json
    intermediate.dsdl
    main-intel.yaml
    news.yaml
    notifications.yaml
    urgent.dsdl
```

### Building the Installer

Navigate to the `src-tauri` directory:
```
cd desktop/tauri/src-tauri
```

Run the following commands to build the installers:

For both NSIS and WIX installers:
```
cargo tauri bundle
```

For NSIS installer only:
```
cargo tauri bundle --bundles nsis
```

For WIX installer only:
```
cargo tauri bundle --bundles wix
```

The produced files will be in:
```
target\release\bundle\msi\
target\release\bundle\nsis\
```

## Debug MSI Installer

To see error messages during the build of the installer, run the bundler with the verbose flag:
```
cargo tauri bundle --bundles msi --verbose
```

To examine the logs during installation, run the installer with the following command:
```
msiexec /i "target\release\bundle\msi\Portmaster_0.1.0_x64_en-US.msi" /lv install.log
```
