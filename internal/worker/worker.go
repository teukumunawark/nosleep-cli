package worker

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"nosleep-cli/internal/consoleui"
	"nosleep-cli/internal/keepawake"
	coresession "nosleep-cli/internal/session"

	"golang.org/x/sys/windows"
)

func RunBackground(duration time.Duration, autoStopAt *time.Time, mode, label string) error {
	store, err := coresession.DefaultStore()
	if err != nil {
		return err
	}

	active, ok, err := coresession.ActiveState(store)
	if err != nil {
		return err
	}
	if ok {
		consoleui.PrintAlreadyRunning(active)
		return nil
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find current executable: %w", err)
	}

	startedAt := time.Now()

	args := []string{
		"start",
		"--background-worker",
		"--started-at", startedAt.Format(time.RFC3339Nano),
		"--session-mode", mode,
		"--mode", label,
	}
	if autoStopAt != nil {
		args = append(args, "--auto-stop-at", autoStopAt.Format(time.RFC3339Nano))
	}

	cmd := exec.Command(executable, args...)
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

	state := coresession.State{
		PID:              cmd.Process.Pid,
		StartedAt:        startedAt,
		ProcessStartedAt: &processStartedAt,
		Mode:             mode,
		AwakeMode:        coresession.AwakeModeSystemDisplay,
		AutoStopAt:       autoStopAt,
		Executable:       executable,
		Label:            label,
	}

	if err := store.Write(state); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release background process: %w", err)
	}

	consoleui.PrintBackgroundStarted(mode, label, autoStopAt, startedAt)

	return nil
}

func RunBackgroundWorker(startedAtStr, autoStopAtStr, mode, label string) error {
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

	waitTimer := time.NewTimer(wait)
	defer waitTimer.Stop()

	select {
	case <-waitTimer.C:
	case <-stop:
	}

	return nil
}
