# NoSleep CLI

<img src="assets/nosleep-logo.avif" alt="NoSleep CLI" width="480">

**Keep your Windows machine awake from the command line.**

<p>
  <a href="https://github.com/Lucu-lucuan-Lab/nosleep-cli/actions/workflows/ci.yml"><img src="https://github.com/Lucu-lucuan-Lab/nosleep-cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/Lucu-lucuan-Lab/nosleep-cli/releases"><img src="https://img.shields.io/github/v/release/Lucu-lucuan-Lab/nosleep-cli?label=release" alt="Latest release"></a>
  <img src="https://img.shields.io/badge/platform-Windows-0078D4" alt="Windows">
  <img src="https://img.shields.io/badge/Go-1.26.2+-00ADD8" alt="Go 1.26.2+">
</p>

<p>
  <code>nosleep start --duration 30m</code>
  &nbsp;|&nbsp;
  <code>nosleep start --background --duration 2h</code>
  &nbsp;|&nbsp;
  <code>nosleep status -w</code>
</p>

---

NoSleep CLI keeps a Windows machine awake for a fixed duration or until you stop
it. It uses the Windows `SetThreadExecutionState` API to keep the system and
display awake without moving the mouse, pressing keys, or simulating user input.

## Quick Install

```powershell
irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nosleep-cli/main/install.ps1 | iex
```

## Install

Run the following command in PowerShell to download and install the latest release automatically:

```powershell
irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nosleep-cli/main/install.ps1 | iex
```

The installer:

- downloads the binary for the current Windows architecture
- verifies the binary with the release SHA-256 checksum
- installs `nosleep.exe` to `%LOCALAPPDATA%\Programs\NoSleep`
- appends the install directory to the User `Path`

Open a new terminal after installation, then verify the command location:

```powershell
where.exe nosleep
```

Expected output:

```text
C:\Users\<you>\AppData\Local\Programs\NoSleep\nosleep.exe
```

Check the installed version:

```powershell
nosleep version
```

## Update

Run the installer again to replace the local binary with the latest release:

```powershell
irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nosleep-cli/main/install.ps1 | iex
```

If NoSleep is running in the background, stop it before updating:

```powershell
nosleep stop
```

## Uninstall

Stop any active session, then run the uninstaller:

```powershell
nosleep stop
irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nosleep-cli/main/uninstall.ps1 | iex
```

The uninstaller removes `nosleep.exe`, removes `%LOCALAPPDATA%\Programs\NoSleep`
from the User `Path`, and removes NoSleep's local state file.

### Manual install

Download the binary for your architecture from the latest release:

- `nosleep-windows-amd64.exe` for most Windows PCs
- `nosleep-windows-arm64.exe` for Windows ARM64

Rename the file to `nosleep.exe`, place it in a directory on your User `Path`,
and verify it against `checksums.txt` from the same release.

## Usage

Start an open-ended session:

```powershell
nosleep start
```

Run for a fixed duration:

```powershell
nosleep start --duration 30m
nosleep start --duration 2h
nosleep start --duration 1h30m
```

Run until a specific 24-hour time:

```powershell
nosleep start --until 17:30
```

Start in the background:

```powershell
nosleep start --background --duration 2h
nosleep status
nosleep stop
```

Show the installed version:

```powershell
nosleep version
```

Attach a label to the session:

```powershell
nosleep start --duration 45m --mode Monitoring
```

Stop the session with `q`, `esc`, or `Ctrl+C`.

## Options

| Flag           | Default   | Description                                        |
|----------------|-----------|----------------------------------------------------|
| `--duration`   | none      | Session duration such as `30m`, `2h`, or `1h30m`.  |
| `--until`      | none      | Auto-stop time in 24-hour format, such as `17:30`. |
| `--background` | `false`   | Start NoSleep without keeping the terminal open.   |
| `--mode`       | `generic` | Optional label displayed for the current session.  |

For compatibility, `nosleep --duration 30m` still starts a foreground session.
Only one background session is allowed at a time.

## Build

Requirements:

- Windows
- Go 1.26.2 or newer

Build from source:

```powershell
go build -o nosleep.exe .\cmd\nosleep
```

Run checks:

```powershell
go test .\...
```

## Release

Releases are built by GitHub Actions from version tags:

```powershell
git tag v0.1.1
git push origin v0.1.1
```

The release workflow runs tests, builds Windows `amd64` and `arm64` binaries,
writes SHA-256 checksums, and attaches the installer and uninstaller scripts to
the GitHub release.
