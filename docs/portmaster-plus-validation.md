# Portmaster Plus 2.2.8 validation

This fork integrates upstream v2.2.8 and the local-only cleanup. The user installed the matching artifacts into an existing official 2.2.8 installation. Read-only post-install verification confirmed that all four installed files match the corrected build and that the running core identifies Portmaster Plus 2.2.8. The live UI serves the updated title and layout assets. The signed driver, native DLL, configuration, and history were preserved.

## Passed

- `go test -short ./spn/access ./spn/captain ./service/firewall ./service ./service/netquery -count=1`
- `go test ./service/updates -run 'TestLocalArtifact|TestManualOnly|TestExcluded|TestValidate' -count=1 -v`
- `node --test desktop/angular/e2e/local-settings.test.cjs`: 2 tests pass.
- Production builds of both Angular libraries and the main Angular application. Main initial bundle: 4.59 MB. Existing CommonJS and advisory 4 MB bundle-budget warnings remain.
- Production tauri-builtin frontend build: initial bundle 204.56 kB.
- Windows x64 core build; version command identifies Portmaster Plus 2.2.8 with portmaster-plus build source.
- Windows x64 native desktop release build; Windows file/product name Portmaster Plus, version 2.2.8.
- `git -c core.whitespace=cr-at-eol diff --check`: no whitespace errors (preserves existing Windows line endings).

Behavior regressions cover local Plus features and record locking, retained stored credentials with local plan publication, zero account transport requests, rejected account mutations, ignored legacy SPN settings, dormant status hooks, disabled firewall tunneling, retained history/bandwidth cleanup, hidden SPN and software-update settings, and enforced manual application updates.

The hashing test measured approximately 33,538 allocated bytes per 8 MiB plain artifact after streaming, versus 8,399,289 before. This concerns temporary startup indexing allocation, not steady-state process RAM.

Shared assets archive decreased from 2,633,080 bytes before removing SPN-only assets to 1,803,018 bytes afterward (approximately 31.5%).

Independent final code review found no critical or important regressions. The final Desktop stage of the Windows build script passed against the finalized sources and built-in UI.

## Limits

- The full updater suite was attempted. Existing update/move/repair tests fail under this Windows user token because production directory ACLs deny write/delete access. Focused new policy and hashing tests pass; production ACL protection was not relaxed.
- The full service-construction regression skips on Windows because service directory setup needs an administrator token. Dormant Captain, account isolation, firewall, and configuration-change checks pass independently.
- Live verification covered service status, all four installed file hashes, runtime build identity, the served UI title, and the corrected main/CSS asset identifiers. Direct UI automation was unavailable; complete interactive DNS/filtering/rules/tray and visual regression coverage is not claimed.
- No before/after steady-state RAM claim is made. The active official core baseline and heap analysis are documented in portmaster-plus-memory.md.

## Artifacts

Matching outputs are under `dist/portmaster-plus/windows_amd64/`: portmaster-core.exe, portmaster.exe, portmaster.zip, assets.zip, and SHA256SUMS. Use the official 2.2.8 signed driver and native DLL. Build instructions and backup/rollback guidance are in README.md and packaging/README.md. The release ZIP is a replacement bundle, not a signed installer.

A combined replacement bundle is available at `dist/portmaster-plus-2.2.8-windows-amd64.zip`, containing only the four matching artifacts and SHA256SUMS. This is not an installer.

## Settings and scrolling correction

The settings update notice now sits below its heading. The document no longer scrolls outside the app shell, and the monitor content pane can shrink and contain its own scrolling. The original monitor/document layout rules were unchanged from upstream before this correction; the notice alignment was introduced by this fork. Production Angular rebuild and both local-settings tests pass. The main UI archive and combined bundle were regenerated; the core, desktop executable, and assets are unchanged. Direct UI inspection was unavailable, so post-install visual verification is still needed.

## Release metadata

The release core is rebuilt from committed source so upstream does not append its dirty-checkout "dev build" suffix. Runtime behavior matches the validated local build. The release tag is `v2.2.8-plus.1`; application manifests remain at the matching upstream version 2.2.8.
