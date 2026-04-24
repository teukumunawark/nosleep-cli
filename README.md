# NoSleep CLI

NoSleep CLI keeps a Windows machine awake for a fixed duration or until you stop
it. It uses the Windows `SetThreadExecutionState` API to keep the system and
display awake without moving the mouse, pressing keys, or simulating user input.

## Install

Download `install.ps1` from the latest release:

https://github.com/teukumunawark/nosleep-cli/releases/latest

Review the script, then run:

```powershell
.\install.ps1 -AddToPath
```

The installer:

- downloads the binary for the current Windows architecture
- verifies the binary with the release SHA-256 checksum
- installs `nosleep.exe` to `%LOCALAPPDATA%\Programs\nosleep`
- appends the install directory to the User `Path` only when `-AddToPath` is set

Open a new terminal after changing `Path`, then verify the command location:

```powershell
where.exe nosleep
```

Expected output:

```text
C:\Users\<you>\AppData\Local\Programs\nosleep\nosleep.exe
```

### Manual install

Download the binary for your architecture from the latest release:

- `nosleep-windows-amd64.exe` for most Windows PCs
- `nosleep-windows-arm64.exe` for Windows ARM64

Rename the file to `nosleep.exe`, place it in a directory on your User `Path`,
and verify it against `checksums.txt` from the same release.

## Usage

Start an open-ended session:

```powershell
nosleep
```

Run for a fixed duration:

```powershell
nosleep -duration 30m
nosleep -duration 2h
```

Attach a label to the session:

```powershell
nosleep -duration 45m -mode Monitoring
```

Stop the session with `q`, `esc`, or `Ctrl+C`.

## Options

| Flag | Default | Description |
| --- | --- | --- |
| `-duration` | `0` | Session duration parsed by Go's `time.ParseDuration`, such as `30m`, `1h`, or `1h30m`. `0` runs until stopped. |
| `-mode` | `generic` | Optional label displayed for the current session. |

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

The repository also includes a convenience build script:

```powershell
.\build.ps1
```

By default, the script writes the binary to `C:\Tools\nosleep\nosleep.exe`.

## Release

Releases are built by GitHub Actions from version tags:

```powershell
git tag v0.1.1
git push origin v0.1.1
```

The release workflow runs tests, builds Windows `amd64` and `arm64` binaries,
writes SHA-256 checksums, and attaches the installer script to the GitHub
release.

## Project Layout

```text
cmd/nosleep/            CLI entry point
internal/keepawake/     Windows keep-awake integration
internal/tui/           Terminal UI
```
