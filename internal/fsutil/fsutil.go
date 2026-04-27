package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func SamePath(a, b string) bool {
	return strings.EqualFold(NormalizePath(a), NormalizePath(b))
}

func NormalizePath(path string) string {
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
