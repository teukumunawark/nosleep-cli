package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"nosleep-cli/internal/consoleui"
	"nosleep-cli/internal/keepawake"
	coresession "nosleep-cli/internal/session"
	"nosleep-cli/internal/timer"
	"nosleep-cli/internal/tui"
	"nosleep-cli/internal/worker"
)

var Version = "dev"

type Session struct {
	Duration   time.Duration
	StartedAt  time.Time
	AutoStopAt *time.Time
	Mode       string
	Label      string
}

func UsagePrintf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

func Run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "start":
			return RunStart(args[1:])
		case "help", "-help", "--help":
			PrintUsage()
			return nil
		case "status":
			return RunStatus(args[1:])
		case "stop":
			return RunStop()
		case "version":
			fmt.Println(VersionString())
			return nil
		}
	}

	return RunStart(args)
}

func VersionString() string {
	return "nosleep " + Version
}

func RunStart(args []string) error {
	flags := flag.NewFlagSet("start", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = PrintUsage

	durationStr := flags.String("duration", "", "Session duration, for example 30m, 2h, or 1h30m.")
	untilStr := flags.String("until", "", "Keep awake until a 24-hour time, for example 17:30.")
	mode := flags.String("mode", "generic", "Optional session label, for example Monitoring or Reading.")
	background := flags.Bool("background", false, "Start NoSleep in the background.")

	backgroundWorker := flags.Bool("background-worker", false, "")
	startedAtStr := flags.String("started-at", "", "")
	autoStopAtStr := flags.String("auto-stop-at", "", "")
	sessionMode := flags.String("session-mode", "", "")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unknown argument: %s", flags.Arg(0))
	}
	if *backgroundWorker {
		return worker.RunBackgroundWorker(*startedAtStr, *autoStopAtStr, *sessionMode, *mode)
	}
	if *durationStr != "" && *untilStr != "" {
		return fmt.Errorf("use either --duration or --until, not both")
	}

	session, err := newSession(*durationStr, *untilStr, *mode, time.Now())
	if err != nil {
		return err
	}
	if *background {
		return worker.RunBackground(session.Duration, session.AutoStopAt, session.Mode, session.Label)
	}

	return runForeground(session)
}

func runForeground(session Session) error {
	if err := keepawake.SetKeepAwake(true); err != nil {
		return fmt.Errorf("enable Windows keep-awake mode: %w", err)
	}
	defer func() {
		_ = keepawake.SetKeepAwake(false)
	}()

	if err := tui.Start(session.tuiSession(false)); err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}

	return nil
}

func (s Session) tuiSession(watch bool) tui.Session {
	var autoStopAt time.Time
	if s.AutoStopAt != nil {
		autoStopAt = *s.AutoStopAt
	}

	return tui.Session{
		Duration:   s.Duration,
		StartedAt:  s.StartedAt,
		AutoStopAt: autoStopAt,
		Kind:       coresession.ModeLabel(s.Mode),
		Label:      s.Label,
		WatchMode:  watch,
	}
}

func RunStatus(args []string) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	watch := flags.Bool("w", false, "Watch status in real-time TUI")
	flags.BoolVar(watch, "watch", false, "Watch status in real-time TUI")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unknown argument: %s", flags.Arg(0))
	}

	store, err := coresession.DefaultStore()
	if err != nil {
		return err
	}

	state, ok, err := coresession.ActiveState(store)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("NoSleep is not running.")
		return nil
	}

	if *watch {
		var duration time.Duration
		if state.AutoStopAt != nil {
			duration = state.AutoStopAt.Sub(state.StartedAt)
		}

		tuiSess := tui.Session{
			Duration:   duration,
			StartedAt:  state.StartedAt,
			AutoStopAt: orZeroTime(state.AutoStopAt),
			Kind:       coresession.ModeLabel(state.Mode),
			Label:      state.Label,
			WatchMode:  true,
		}

		return tui.Start(tuiSess)
	}

	consoleui.PrintStatus(state, time.Now())
	return nil
}

func orZeroTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func RunStop() error {
	store, err := coresession.DefaultStore()
	if err != nil {
		return err
	}

	state, ok, err := coresession.ActiveState(store)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("NoSleep is not running.")
		return nil
	}

	if err := coresession.KillMatchingProcess(state.PID, state.Executable, state.ProcessStartedAt); err != nil && !errors.Is(err, coresession.ErrProcessNotRunning) {
		return err
	}
	if err := store.Remove(); err != nil {
		return err
	}

	fmt.Print(consoleui.StoppedOutput())
	return nil
}

func newSession(durationStr, untilStr, label string, now time.Time) (Session, error) {
	switch {
	case durationStr != "":
		duration, err := timer.ParseDuration(durationStr)
		if err != nil {
			return Session{}, invalidDurationError(durationStr)
		}
		autoStopAt := now.Add(duration)
		return Session{
			Duration:   duration,
			StartedAt:  now,
			AutoStopAt: &autoStopAt,
			Mode:       coresession.ModeTimed,
			Label:      label,
		}, nil
	case untilStr != "":
		autoStopAt, err := timer.ParseUntil(now, untilStr)
		if err != nil {
			return Session{}, invalidUntilError(untilStr)
		}
		return Session{
			StartedAt:  now,
			AutoStopAt: &autoStopAt,
			Mode:       coresession.ModeUntil,
			Label:      label,
		}, nil
	default:
		return Session{
			StartedAt: now,
			Mode:      coresession.ModeOpenEnded,
			Label:     label,
		}, nil
	}
}

func invalidDurationError(value string) error {
	return fmt.Errorf(`Invalid duration: %s

Use examples:
  nosleep start --duration 30m
  nosleep start --duration 2h
  nosleep start --duration 1h30m`, value)
}

func invalidUntilError(value string) error {
	return fmt.Errorf(`Invalid time: %s

Use 24-hour format:
  nosleep start --until 17:30`, value)
}

func PrintUsage() {
	UsagePrintf("NoSleep CLI - Windows sleep prevention utility\n\n")
	UsagePrintf("Keeps the system and display awake without simulating mouse or keyboard input.\n\n")
	UsagePrintf("Usage:\n")
	UsagePrintf("  nosleep start [flags]\n")
	UsagePrintf("  nosleep [flags]\n")
	UsagePrintf("  nosleep status [-w]\n")
	UsagePrintf("  nosleep stop\n")
	UsagePrintf("  nosleep version\n\n")
	UsagePrintf("Flags for start:\n")
	UsagePrintf("  --duration value   Session duration, for example 30m, 2h, or 1h30m.\n")
	UsagePrintf("  --until value      Keep awake until a 24-hour time, for example 17:30.\n")
	UsagePrintf("  --background       Start NoSleep in the background.\n")
	UsagePrintf("  --mode value       Optional session label, for example Monitoring or Reading.\n\n")
	UsagePrintf("Flags for status:\n")
	UsagePrintf("  -w, --watch        Watch the session in real-time TUI.\n\n")
	UsagePrintf("Examples:\n")
	UsagePrintf("  nosleep start\n")
	UsagePrintf("  nosleep start --duration 30m\n")
	UsagePrintf("  nosleep start --until 17:30\n\n")
	UsagePrintf("  nosleep start --background --duration 2h\n")
	UsagePrintf("  nosleep status -w\n")
	UsagePrintf("  nosleep stop\n")
	UsagePrintf("  nosleep version\n\n")
	UsagePrintf("Notes:\n")
	UsagePrintf("  - Press Q, ESC, or Ctrl+C to stop the session.\n")
	UsagePrintf("  - Uses the Windows SetThreadExecutionState API.\n")
}
