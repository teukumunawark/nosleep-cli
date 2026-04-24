# NoSleep CLI

NoSleep CLI keeps a Windows machine awake for a fixed duration or until you stop it.

It calls the native Windows `SetThreadExecutionState` API to prevent system and display sleep. It does not move the
mouse, press keys, or simulate user input.

## Status

- Platform: Windows
- Runtime dependencies: none
- Build requirement: Go 1.26.2 or newer

## Installation

Download `install.ps1` from the latest GitHub release:

https://github.com/teukumunawark/nosleep-cli/releases/latest

Review the script, then run it from PowerShell:

```powershell
.\install.ps1
```

The installer downloads the matching Windows binary for the current machine,
verifies it with the release SHA-256 checksum, and installs it for the current
user:

```text
%LOCALAPPDATA%\Programs\nosleep\nosleep.exe
```

To also add the install directory to the User `Path`, run:

```powershell
.\install.ps1 -AddToPath
```

The script appends the install directory only if it is not already present.
Open a new terminal after changing `Path`.

To set up `Path` manually instead, add this directory to the User `Path`
environment variable:

```text
%LOCALAPPDATA%\Programs\nosleep
```

One way to do this on Windows is to open **Edit environment variables for your
account**, edit the User `Path` variable, and add
`%LOCALAPPDATA%\Programs\nosleep` as a new entry.

Verify that Windows resolves `nosleep` from the expected location:

```powershell
where.exe nosleep
```

Expected output:

```text
C:\Users\<you>\AppData\Local\Programs\nosleep\nosleep.exe
```

### Manual install

Download `nosleep-windows-amd64.exe` for most Windows PCs, or
`nosleep-windows-arm64.exe` on Windows ARM64. Rename the downloaded file to
`nosleep.exe`, place it in a directory on your User `Path`, and verify the
download against `checksums.txt` from the same release.

### Build from source

Build directly with Go:

```powershell
go build -o nosleep.exe .\cmd\nosleep
```

The repository also includes a convenience build script:

```powershell
.\build.ps1
```

By default, the script writes the binary to `C:\Tools\nosleep\nosleep.exe`.

## Usage

Start an open-ended session:

```powershell
nosleep
```

Run for a specific duration:

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

| Flag        | Default   | Description                                                                                                    |
|-------------|-----------|----------------------------------------------------------------------------------------------------------------|
| `-duration` | `0`       | Session duration parsed by Go's `time.ParseDuration`, such as `30m`, `1h`, or `1h30m`. `0` runs until stopped. |
| `-mode`     | `generic` | Optional label displayed for the current session.                                                              |

## Development

Run checks:

```powershell
go test .\...
```

Build the CLI:

```powershell
go build -o nosleep.exe .\cmd\nosleep
```

Create a release:

```powershell
git tag v0.1.0
git push origin v0.1.0
```

Pushing a `v*` tag starts the release workflow. The workflow builds Windows
amd64 and arm64 binaries, writes SHA-256 checksums, and attaches them to a
GitHub release.

Project layout:

```text
cmd/nosleep/            CLI entry point
internal/keepawake/     Windows keep-awake integration
internal/tui/           Terminal UI
```

## Behavior

NoSleep enables `ES_CONTINUOUS`, `ES_SYSTEM_REQUIRED`, and `ES_DISPLAY_REQUIRED` when a session starts. On normal exit,
it restores the execution state with `ES_CONTINUOUS`.
