# safing-portmaster-plus

An unofficial fork of [Safing's Portmaster](https://github.com/safing/portmaster) that enables **Network History** and **Bandwidth Visibility** — the two device-local features otherwise reserved for the paid Portmaster Plus tier — in a self-compiled build.

> **Not affiliated with Safing ICS Technologies GmbH.** "Safing" and "Portmaster" are their trademarks, used here only to describe what this fork modifies. This is a community modification. Do not report issues with this build to Safing.

## What this does

| Feature | Status | Why |
|---|---|---|
| Network History | Enabled | Writes connections to a local SQLite database on your own machine |
| Bandwidth Visibility | Enabled | Counts bytes per connection, locally |
| SPN (Safing Privacy Network) | **Not enabled — and cannot be** | See below |

### Why these two can be enabled and SPN cannot

Portmaster gates features in two fundamentally different ways.

Network History and Bandwidth Visibility are gated by a **boolean check inside the process running on your own machine**. The code implementing them ships complete in every build; a subscription check simply sets a flag in front of it. That is a licensing decision, not a technical one, and this fork changes it.

SPN is gated by **cryptography running on Safing's servers**. Access tokens are blind-signed by their account server and verified by every Hub before it will grant the permissions needed to build a tunnel. That check does not run on your computer, so no patch to this codebase can affect it. Adding SPN to the allowlist would unlock a button and produce nothing but failed connections — which is why the code explicitly refuses to.

## What changed

Three commits, **203 lines added, 0 removed**. Nothing upstream is rewritten or deleted, so this rebases cleanly onto future releases.

- **`spn/access/local_features.go`** *(new)* — an explicit allowlist of locally enabled features, plus the logic that publishes it.
- **`spn/access/database.go`** *(+5)* — `UserRecord.MayUse` consults the allowlist. This wrapper, not `account.User.MayUse`, is the effective gate: it returns `false` early when nobody is logged in, before the real subscription logic is ever reached.
- **`spn/access/module.go`** *(+10)* — applies the feature plan from a worker at startup.
- **`spn/access/local_features_test.go`** *(new)* — regression tests.

### The backend is only half of it

The user interface never calls `MayUse`. It watches the database record `core:spn/account/user` over a websocket and derives every lock — ribbons, disabled toggles, hidden data tiles — from `current_plan.feature_ids`. Patching only the backend makes the features *work* while the UI keeps them invisible and unclickable.

So this fork also publishes the same allowlist through that record. The record is stored in the **logged-out** state: `IsLoggedIn()` still reports false, nothing contacts Safing, and logging in for real replaces it wholesale.

## Installing

You keep the official installation and replace exactly one file: the userspace Go binary. The signed kernel driver, the UI bundle, and the service registration all stay as the installer left them. No Windows Driver Kit, no test-signing, no Secure Boot changes.

> **Read this first.** Portmaster is a firewall with a kernel driver. If the core service fails to start, the driver keeps intercepting packets with no userspace left to make decisions, and **all network traffic stops** — including after a reboot. The `Manual` startup step below exists so a failure cannot survive a restart. Do not skip it.

1. Install Portmaster **v2.2.1** normally from [safing.io](https://safing.io/download/) and confirm it works.

2. Verify the version matches this build, in an **elevated** PowerShell:

   ```powershell
   & "C:\Program Files\Portmaster\portmaster-core.exe" version
   ```

   If it is not 2.2.1, do not continue — build from the matching tag instead.

3. Make a failure survivable:

   ```powershell
   Set-Service PortmasterCore -StartupType Manual
   ```

4. Swap the binary:

   ```powershell
   Stop-Service PortmasterCore
   Copy-Item "C:\Program Files\Portmaster\portmaster-core.exe" "C:\Program Files\Portmaster\portmaster-core.exe.orig"
   Copy-Item "<downloaded>\portmaster-core.exe" "C:\Program Files\Portmaster\portmaster-core.exe" -Force
   Start-Service PortmasterCore
   ```

5. If it starts promptly, restore automatic startup:

   ```powershell
   Set-Service PortmasterCore -StartupType Automatic
   ```

   If it hangs instead, reboot. With `Manual` startup the service will not come up, your network works normally, and you can restore `portmaster-core.exe.orig` at your leisure.

6. Open Portmaster. The dashboard should read **"Your current plan is Local Build"**, and Network History and Bandwidth Visibility should be unlocked. Enable Network History from its toggle in Global Settings.

### Reverting

Stop the service, copy `portmaster-core.exe.orig` back over `portmaster-core.exe`, start it. Note that the stored user record persists: the stock UI will still show the features unlocked while the stock backend refuses them. Logging in, or clearing the record, resets it.

## Building from source

Requires Go 1.24+. No C toolchain — the build is `CGO_ENABLED=0`.

```bash
git clone https://github.com/maxdorx/safing-portmaster-plus.git
cd safing-portmaster-plus
CGO_ENABLED=0 go build -trimpath \
  -ldflags="-X github.com/safing/portmaster/base/info.version=2.2.1" \
  -o portmaster-core.exe ./cmds/portmaster-core
```

Run the tests:

```bash
go test ./spn/access/ -short
```

`TestApplyLocalFeaturesReleasesLock` is not optional decoration. An earlier version of this patch held a record lock across a save that re-locks the same record; `sync.Mutex` is not reentrant, so it self-deadlocked inside module startup and took down all networking on the test machine until Portmaster was uninstalled. That test fails if the locking discipline regresses.

## License and trademarks

GPLv3, same as upstream — see [LICENSE](LICENSE). Distributing binaries built from this source obliges you to make the corresponding source available; that is what this repository is for.

"Safing", "Portmaster" and their logos are trademarks of Safing ICS Technologies GmbH. See [TRADEMARKS](TRADEMARKS). This fork uses those names descriptively and claims no affiliation, endorsement, or origin.

## Support the upstream project

Portmaster is a genuinely good piece of open-source software from a small team, and the subscription is what funds it. If you get value out of it, [buy a package](https://safing.io/pricing/) — especially if you want SPN, which is the part with real server and bandwidth costs behind it and the part this fork cannot give you.

Upstream README preserved at [README.upstream.md](README.upstream.md).
