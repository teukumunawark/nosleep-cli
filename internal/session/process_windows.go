package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const processStatusStillActive = 259

var ErrProcessNotRunning = errors.New("process is not running")

func ProcessStartedAt(pid int) (time.Time, error) {
	handle, err := openProcess(pid, windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE)
	if err != nil {
		return time.Time{}, err
	}
	defer windows.CloseHandle(handle)

	return processStartedAt(handle)
}

func ProcessMatches(pid int, executable string, startedAt *time.Time) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	handle, err := openProcess(pid, windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE)
	if err != nil {
		return false, nil
	}
	defer windows.CloseHandle(handle)

	return processMatchesHandle(handle, executable, startedAt)
}

func KillMatchingProcess(pid int, executable string, startedAt *time.Time) error {
	const rights = windows.PROCESS_QUERY_LIMITED_INFORMATION | windows.PROCESS_TERMINATE | windows.SYNCHRONIZE

	handle, err := openProcess(pid, rights)
	if err != nil {
		return ErrProcessNotRunning
	}
	defer windows.CloseHandle(handle)

	matches, err := processMatchesHandle(handle, executable, startedAt)
	if err != nil {
		return err
	}
	if !matches {
		return ErrProcessNotRunning
	}

	if err := windows.TerminateProcess(handle, 0); err != nil {
		return fmt.Errorf("terminate process: %w", err)
	}

	return nil
}

func processMatchesHandle(handle windows.Handle, executable string, startedAt *time.Time) (bool, error) {
	running, err := processIsRunning(handle)
	if err != nil {
		return false, err
	}
	if !running {
		return false, nil
	}

	if strings.TrimSpace(executable) == "" {
		return false, nil
	}

	imagePath, err := processImagePath(handle)
	if err != nil {
		return false, fmt.Errorf("query process image path: %w", err)
	}
	if !samePath(imagePath, executable) {
		return false, nil
	}

	if startedAt == nil || startedAt.IsZero() {
		return false, nil
	}
	actualStartedAt, err := processStartedAt(handle)
	if err != nil {
		return false, fmt.Errorf("query process start time: %w", err)
	}
	if !sameProcessStart(actualStartedAt, *startedAt) {
		return false, nil
	}

	return true, nil
}

func processImagePath(handle windows.Handle) (string, error) {
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

func processIsRunning(handle windows.Handle) (bool, error) {
	result, err := windows.WaitForSingleObject(handle, 0)
	if err != nil {
		if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return processExitCodeIsRunning(handle)
		}
		return false, fmt.Errorf("wait for process: %w", err)
	}
	return result == uint32(windows.WAIT_TIMEOUT), nil
}

func processExitCodeIsRunning(handle windows.Handle) (bool, error) {
	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false, fmt.Errorf("get process exit code: %w", err)
	}
	return exitCode == processStatusStillActive, nil
}

func processStartedAt(handle windows.Handle) (time.Time, error) {
	var createdAt, exitedAt, kernelTime, userTime windows.Filetime
	if err := windows.GetProcessTimes(handle, &createdAt, &exitedAt, &kernelTime, &userTime); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, createdAt.Nanoseconds()).UTC(), nil
}

func openProcess(pid int, rights uint32) (windows.Handle, error) {
	if pid <= 0 {
		return 0, ErrProcessNotRunning
	}

	handle, err := windows.OpenProcess(rights, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	return handle, nil
}

func samePath(a, b string) bool {
	return strings.EqualFold(normalizePath(a), normalizePath(b))
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	} else if errors.Is(err, os.ErrNotExist) {
		path = filepath.Clean(path)
	}

	return filepath.Clean(path)
}

func sameProcessStart(a, b time.Time) bool {
	return a.UTC().Equal(b.UTC())
}
