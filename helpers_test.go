package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn and returns everything written to os.Stdout during that call.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String()
}

// tempFile creates a temporary file with the given content and registers cleanup.
func tempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "rh_test_*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// readHashesFor calls readFile on path and returns the hash for each line in order.
// It requires setupCache(t) to have been called first so NextID starts at 0.
func readHashesFor(t *testing.T, path string) []string {
	t.Helper()
	var hashes []string
	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})
	if strings.TrimSpace(out) == "" {
		return hashes
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 1 && parts[0] != "" {
			hashes = append(hashes, parts[0])
		}
	}
	return hashes
}
