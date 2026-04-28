package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "spaces only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "absolute path",
			input:    filepath.Join(wd, "testdir", "file.txt"),
			expected: filepath.Join(wd, "testdir", "file.txt"),
		},
		{
			name:     "relative path",
			input:    filepath.Join(".", "testdir", "file.txt"),
			expected: filepath.Join(wd, "testdir", "file.txt"),
		},
		{
			name:     "path with dot dot",
			input:    filepath.Join(wd, "testdir", "..", "file.txt"),
			expected: filepath.Join(wd, "file.txt"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizePath(tc.input)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestNormalizePath_Symlink(t *testing.T) {
	tmpDir := t.TempDir()

	targetFile := filepath.Join(tmpDir, "target.txt")
	err := os.WriteFile(targetFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	symlinkFile := filepath.Join(tmpDir, "symlink.txt")
	err = os.Symlink(targetFile, symlinkFile)
	if err != nil {
		t.Skipf("skipping symlink test, symlink creation failed (requires admin on Windows): %v", err)
	}

	got := NormalizePath(symlinkFile)
	expected := NormalizePath(targetFile)

	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestSamePath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "exact same absolute path",
			a:        filepath.Join(wd, "file.txt"),
			b:        filepath.Join(wd, "file.txt"),
			expected: true,
		},
		{
			name:     "different case",
			a:        filepath.Join(wd, "File.txt"),
			b:        filepath.Join(wd, "file.txt"),
			expected: true,
		},
		{
			name:     "relative and absolute",
			a:        "file.txt",
			b:        filepath.Join(wd, "file.txt"),
			expected: true,
		},
		{
			name:     "different paths",
			a:        filepath.Join(wd, "file1.txt"),
			b:        filepath.Join(wd, "file2.txt"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SamePath(tc.a, tc.b)
			if got != tc.expected {
				t.Errorf("SamePath(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}
