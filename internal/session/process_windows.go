package session

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

const stillActive = 259

var ErrProcessNotRunning = errors.New("process is not running")

func ProcessMatches(pid int, executable string) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false, nil
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false, fmt.Errorf("get process exit code: %w", err)
	}
	if exitCode != stillActive {
		return false, nil
	}

	if strings.TrimSpace(executable) == "" {
		return true, nil
	}

	imagePath, err := processImagePath(handle)
	if err != nil {
		return false, fmt.Errorf("query process image path: %w", err)
	}
	if !samePath(imagePath, executable) {
		return false, nil
	}

	return true, nil
}

func KillMatchingProcess(pid int, executable string) error {
	matches, err := ProcessMatches(pid, executable)
	if err != nil {
		return err
	}
	if !matches {
		return ErrProcessNotRunning
	}

	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("open process for termination: %w", err)
	}
	defer windows.CloseHandle(handle)

	if err := windows.TerminateProcess(handle, 0); err != nil {
		return fmt.Errorf("terminate process: %w", err)
	}

	return nil
}

func processImagePath(handle windows.Handle) (string, error) {
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
