package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nosleep-cli/internal/keepawake"
	coresession "nosleep-cli/internal/session"
	"nosleep-cli/internal/timer"
	"nosleep-cli/internal/tui"

	"golang.org/x/sys/windows"
)

type startSession struct {
	Duration   time.Duration
	AutoStopAt *time.Time
	Mode       string
	Label      string
}

type outputRow struct {
	Label string
	Value string
}

func usagePrintf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "start":
			return runStart(args[1:])
		case "help", "-help", "--help":
			printUsage()
			return nil
		case "status":
			return runStatus()
		case "stop":
			return runStop()
		}
	}

	return runStart(args)
}

func runStart(args []string) error {
	flags := flag.NewFlagSet("start", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = printUsage

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
		return runBackgroundWorker(*startedAtStr, *autoStopAtStr, *sessionMode, *mode)
	}
	if *durationStr != "" && *untilStr != "" {
		return fmt.Errorf("use either --duration or --until, not both")
	}

	session, err := newSession(*durationStr, *untilStr, *mode, time.Now())
	if err != nil {
		return err
	}
	if *background {
		return runBackground(session)
	}

	return runForeground(session)
}

func runForeground(session startSession) error {
	if err := keepawake.SetKeepAwake(true); err != nil {
		return fmt.Errorf("enable Windows keep-awake mode: %w", err)
	}
	defer func() {
		_ = keepawake.SetKeepAwake(false)
	}()

	if err := tui.Start(session.tuiSession()); err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}

	return nil
}

func runBackground(session startSession) error {
	store, err := coresession.DefaultStore()
	if err != nil {
		return err
	}

	active, ok, err := activeState(store)
	if err != nil {
		return err
	}
	if ok {
		printAlreadyRunning(active)
		return nil
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find current executable: %w", err)
	}

	startedAt := time.Now()
	cmd := exec.Command(executable, session.backgroundArgs(startedAt)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start background process: %w", err)
	}

	processStartedAt, err := coresession.ProcessStartedAt(cmd.Process.Pid)
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("query background process start time: %w", err)
	}

	state := session.state(cmd.Process.Pid, startedAt, processStartedAt, executable)
	if err := store.Write(state); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release background process: %w", err)
	}

	fmt.Print(backgroundStartedOutput(session, startedAt))

	return nil
}

func runBackgroundWorker(startedAtStr, autoStopAtStr, mode, label string) error {
	startedAt, err := time.Parse(time.RFC3339Nano, startedAtStr)
	if err != nil {
		return fmt.Errorf("parse background worker start time: %w", err)
	}

	var autoStopAt *time.Time
	if autoStopAtStr != "" {
		parsed, err := time.Parse(time.RFC3339Nano, autoStopAtStr)
		if err != nil {
			return fmt.Errorf("parse background worker auto-stop time: %w", err)
		}
		autoStopAt = &parsed
	}
	if mode == "" {
		mode = coresession.ModeOpenEnded
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find current executable: %w", err)
	}
	store, err := coresession.DefaultStore()
	if err != nil {
		return err
	}

	processStartedAt, err := coresession.ProcessStartedAt(os.Getpid())
	if err != nil {
		return fmt.Errorf("query background worker process start time: %w", err)
	}

	state := coresession.State{
		PID:              os.Getpid(),
		StartedAt:        startedAt,
		ProcessStartedAt: &processStartedAt,
		Mode:             mode,
		AwakeMode:        coresession.AwakeModeSystemDisplay,
		AutoStopAt:       autoStopAt,
		Executable:       executable,
		Label:            label,
	}
	if err := store.Write(state); err != nil {
		return err
	}
	defer func() {
		_ = store.Remove()
	}()

	if err := keepawake.SetKeepAwake(true); err != nil {
		return fmt.Errorf("enable Windows keep-awake mode: %w", err)
	}
	defer func() {
		_ = keepawake.SetKeepAwake(false)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	defer signal.Stop(stop)

	if autoStopAt == nil {
		<-stop
		return nil
	}

	wait := time.Until(*autoStopAt)
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-stop:
	}

	return nil
}

func runStatus() error {
	store, err := coresession.DefaultStore()
	if err != nil {
		return err
	}

	state, ok, err := activeState(store)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("NoSleep is not running.")
		return nil
	}

	printStatus(state, time.Now())
	return nil
}

func runStop() error {
	store, err := coresession.DefaultStore()
	if err != nil {
		return err
	}

	state, ok, err := activeState(store)
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

	fmt.Print(stoppedOutput())
	return nil
}

func activeState(store coresession.Store) (coresession.State, bool, error) {
	state, ok, err := store.Read()
	if err != nil {
		if errors.Is(err, coresession.ErrInvalidState) {
			if removeErr := store.Remove(); removeErr != nil {
				return coresession.State{}, false, removeErr
			}
			return coresession.State{}, false, nil
		}
		return coresession.State{}, false, err
	}
	if !ok {
		return coresession.State{}, false, nil
	}

	matches, err := coresession.ProcessMatches(state.PID, state.Executable, state.ProcessStartedAt)
	if err != nil {
		return coresession.State{}, false, err
	}
	if !matches {
		if err := store.Remove(); err != nil {
			return coresession.State{}, false, err
		}
		return coresession.State{}, false, nil
	}

	return state, true, nil
}

func newSession(durationStr, untilStr, label string, now time.Time) (startSession, error) {
	switch {
	case durationStr != "":
		duration, err := timer.ParseDuration(durationStr)
		if err != nil {
			return startSession{}, invalidDurationError(durationStr)
		}
		autoStopAt := now.Add(duration)
		return startSession{
			Duration:   duration,
			AutoStopAt: &autoStopAt,
			Mode:       coresession.ModeTimed,
			Label:      label,
		}, nil
	case untilStr != "":
		autoStopAt, err := timer.ParseUntil(now, untilStr)
		if err != nil {
			return startSession{}, invalidUntilError(untilStr)
		}
		return startSession{
			AutoStopAt: &autoStopAt,
			Mode:       coresession.ModeUntil,
			Label:      label,
		}, nil
	default:
		return startSession{
			Mode:  coresession.ModeOpenEnded,
			Label: label,
		}, nil
	}
}

func (s startSession) tuiSession() tui.Session {
	var autoStopAt time.Time
	if s.AutoStopAt != nil {
		autoStopAt = *s.AutoStopAt
	}

	return tui.Session{
		Duration:   s.Duration,
		AutoStopAt: autoStopAt,
		Kind:       coresession.ModeLabel(s.Mode),
		Label:      s.Label,
	}
}

func (s startSession) backgroundArgs(startedAt time.Time) []string {
	args := []string{
		"start",
		"--background-worker",
		"--started-at", startedAt.Format(time.RFC3339Nano),
		"--session-mode", s.Mode,
		"--mode", s.Label,
	}
	if s.AutoStopAt != nil {
		args = append(args, "--auto-stop-at", s.AutoStopAt.Format(time.RFC3339Nano))
	}
	return args
}

func (s startSession) state(pid int, startedAt time.Time, processStartedAt time.Time, executable string) coresession.State {
	return coresession.State{
		PID:              pid,
		StartedAt:        startedAt,
		ProcessStartedAt: &processStartedAt,
		Mode:             s.Mode,
		AwakeMode:        coresession.AwakeModeSystemDisplay,
		AutoStopAt:       s.AutoStopAt,
		Executable:       executable,
		Label:            s.Label,
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

func printAlreadyRunning(state coresession.State) {
	fmt.Print(alreadyRunningOutput(state, time.Now()))
}

func printStatus(state coresession.State, now time.Time) {
	fmt.Print(statusOutput(state, now))
}

func backgroundStartedOutput(session startSession, startedAt time.Time) string {
	rows := []outputRow{
		{Label: "Status", Value: "Active in background"},
		{Label: "Mode", Value: coresession.ModeLabel(session.Mode)},
	}
	if session.AutoStopAt == nil {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: "None"})
	} else {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: formatClock(*session.AutoStopAt, startedAt)})
	}

	return renderOutput("NoSleep started", rows, []string{"nosleep status", "nosleep stop"}, nil)
}

func alreadyRunningOutput(state coresession.State, now time.Time) string {
	rows := []outputRow{
		{Label: "Status", Value: "Already running"},
		{Label: "Mode", Value: coresession.ModeLabel(state.Mode)},
	}
	if state.AutoStopAt != nil {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: formatClock(*state.AutoStopAt, now)})
	}

	return renderOutput(
		"NoSleep is already running",
		rows,
		[]string{"nosleep status", "nosleep stop"},
		[]string{"Stop the active session before starting another one."},
	)
}

func statusOutput(state coresession.State, now time.Time) string {
	rows := []outputRow{
		{Label: "Status", Value: "Active"},
		{Label: "Mode", Value: coresession.ModeLabel(state.Mode)},
		{Label: "Started", Value: formatClock(state.StartedAt, now)},
		{Label: "Elapsed", Value: formatDuration(now.Sub(state.StartedAt))},
	}
	if state.AutoStopAt != nil {
		rows = append(rows,
			outputRow{Label: "Remaining", Value: formatDuration(state.AutoStopAt.Sub(now))},
			outputRow{Label: "Auto-stop", Value: formatClock(*state.AutoStopAt, now)},
		)
	} else {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: "None"})
	}
	rows = append(rows,
		outputRow{Label: "Awake", Value: coresession.AwakeModeLabel(state.AwakeMode)},
		outputRow{Label: "PID", Value: fmt.Sprintf("%d", state.PID)},
	)

	return renderOutput("NoSleep status", rows, []string{"nosleep stop"}, nil)
}

func stoppedOutput() string {
	return renderOutput(
		"NoSleep stopped",
		nil,
		nil,
		[]string{"Normal Windows sleep behavior restored."},
	)
}

func renderOutput(title string, rows []outputRow, commands []string, notes []string) string {
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n")

	if len(rows) > 0 {
		b.WriteString("\n")
		for _, row := range rows {
			b.WriteString(fmt.Sprintf("  %-10s %s\n", row.Label, row.Value))
		}
	}

	if len(notes) > 0 {
		b.WriteString("\n")
		for _, note := range notes {
			b.WriteString("  ")
			b.WriteString(note)
			b.WriteString("\n")
		}
	}

	if len(commands) > 0 {
		b.WriteString("\n")
		b.WriteString("Next:\n")
		for _, command := range commands {
			b.WriteString("  ")
			b.WriteString(command)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func formatClock(t time.Time, reference time.Time) string {
	if sameDate(t, reference) {
		return t.Format("15:04:05")
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func printUsage() {
	usagePrintf("NoSleep CLI - Windows sleep prevention utility\n\n")
	usagePrintf("Keeps the system and display awake without simulating mouse or keyboard input.\n\n")
	usagePrintf("Usage:\n")
	usagePrintf("  nosleep start [flags]\n")
	usagePrintf("  nosleep [flags]\n")
	usagePrintf("  nosleep status\n")
	usagePrintf("  nosleep stop\n\n")
	usagePrintf("Flags for start:\n")
	usagePrintf("  --duration value   Session duration, for example 30m, 2h, or 1h30m.\n")
	usagePrintf("  --until value      Keep awake until a 24-hour time, for example 17:30.\n")
	usagePrintf("  --background       Start NoSleep in the background.\n")
	usagePrintf("  --mode value       Optional session label, for example Monitoring or Reading.\n\n")
	usagePrintf("Examples:\n")
	usagePrintf("  nosleep start\n")
	usagePrintf("  nosleep start --duration 30m\n")
	usagePrintf("  nosleep start --until 17:30\n\n")
	usagePrintf("  nosleep start --background --duration 2h\n")
	usagePrintf("  nosleep status\n")
	usagePrintf("  nosleep stop\n\n")
	usagePrintf("Notes:\n")
	usagePrintf("  - Press Q, ESC, or Ctrl+C to stop the session.\n")
	usagePrintf("  - Uses the Windows SetThreadExecutionState API.\n")
}
