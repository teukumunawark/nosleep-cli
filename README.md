# NoSleep CLI

NoSleep CLI keeps a Windows machine awake for a fixed duration or until you stop it.

It calls the native Windows `SetThreadExecutionState` API to prevent system and display sleep. It does not move the
mouse, press keys, or simulate user input.

## Status

- Platform: Windows
- Runtime dependencies: none
- Build requirement: Go 1.26.2 or newer

## Installation

Build from source:

```powershell
go build -o nosleep.exe .\cmd\nosleep
```

The repository also includes a convenience build script:

```powershell
.\build.ps1
```

By default, the script writes the binary to:

```text
C:\Tools\nosleep\nosleep.exe
```

To run `nosleep` from any directory, add the build directory to the User
`Path` environment variable:

```text
C:\Tools\nosleep
```

One way to do this on Windows is to open **Edit environment variables for your
account**, edit the User `Path` variable, and add `C:\Tools\nosleep` as a new
entry. Open a new terminal after changing `Path`.

Verify that Windows resolves `nosleep` from the expected location:

```powershell
where.exe nosleep
```

Expected output:

```text
C:\Tools\nosleep\nosleep.exe
```

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

Project layout:

```text
cmd/nosleep/            CLI entry point
internal/keepawake/     Windows keep-awake integration
internal/tui/           Terminal UI
```

## Behavior

NoSleep enables `ES_CONTINUOUS`, `ES_SYSTEM_REQUIRED`, and `ES_DISPLAY_REQUIRED` when a session starts. On normal exit,
it restores the execution state with `ES_CONTINUOUS`.
