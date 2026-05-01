package main

import (
	"strings"
	"testing"
)

func TestReadFile_Basic(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\ngamma\n")

	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d: %q", len(lines), out)
	}

	wantContent := []string{"alpha", "beta", "gamma"}
	for i, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			t.Errorf("line %d: expected 'hash content', got %q", i, line)
			continue
		}
		if parts[0] != indexToHash(i) {
			t.Errorf("line %d: expected hash %s, got %s", i, indexToHash(i), parts[0])
		}
		if parts[1] != wantContent[i] {
			t.Errorf("line %d: expected content %q, got %q", i, wantContent[i], parts[1])
		}
	}
}

func TestReadFile_Empty(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "")

	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})

	if out != "" {
		t.Errorf("expected empty output for empty file, got %q", out)
	}
}

func TestReadFile_SingleLine(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "only line\n")

	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 output line, got %d: %q", len(lines), out)
	}
	parts := strings.SplitN(lines[0], " ", 2)
	if len(parts) != 2 || parts[1] != "only line" {
		t.Errorf("unexpected output: %q", lines[0])
	}
}

func TestReadFile_NotFound(t *testing.T) {
	err := readFile("/nonexistent/path/rh_missing.txt")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestReadFile_HashesAreUnique(t *testing.T) {
	setupCache(t)
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString("line\n")
	}
	path := tempFile(t, sb.String())

	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})

	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		hash := strings.SplitN(line, " ", 2)[0]
		if seen[hash] {
			t.Errorf("duplicate hash in output: %s", hash)
		}
		seen[hash] = true
	}
}

func TestReadFile_HashesAreLowercaseAlpha(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "a\nb\nc\n")

	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})

	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		hash := strings.SplitN(line, " ", 2)[0]
		if len(hash) != 4 {
			t.Errorf("hash %q is not exactly 4 chars", hash)
		}
		for _, c := range hash {
			if c < 'a' || c > 'z' {
				t.Errorf("hash %q contains non-lowercase character %c", hash, c)
			}
		}
	}
}

func TestReadFile_HashMatchesIndexToHash(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "x\ny\nz\nw\n")

	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})

	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		got := strings.SplitN(line, " ", 2)[0]
		want := indexToHash(i)
		if got != want {
			t.Errorf("line %d: expected hash %s, got %s", i, want, got)
		}
	}
}

func TestReadFile_HashesPersistAcrossReads(t *testing.T) {
	// Reading a file twice must return the same hashes both times.
	setupCache(t)
	path := tempFile(t, "foo\nbar\nbaz\n")

	first := readHashesFor(t, path)
	second := readHashesFor(t, path)

	if len(first) != len(second) {
		t.Fatalf("hash count changed between reads: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("line %d hash changed between reads: %s → %s", i, first[i], second[i])
		}
	}
}
