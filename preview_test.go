package main

import (
	"os"
	"strings"
	"testing"
)

func TestPreviewLines_Basic(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\ngamma\ndelta\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := previewLines(path, hashes[1], hashes[2]); err != nil {
			t.Fatalf("previewLines: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(lines), out)
	}
	cases := []struct {
		hash    string
		content string
	}{
		{hashes[1], "beta"},
		{hashes[2], "gamma"},
	}
	for i, tc := range cases {
		parts := strings.SplitN(lines[i], " ", 2)
		if len(parts) != 2 {
			t.Errorf("line %d: bad format %q", i, lines[i])
			continue
		}
		if parts[0] != tc.hash {
			t.Errorf("line %d: expected hash %s, got %s", i, tc.hash, parts[0])
		}
		if parts[1] != tc.content {
			t.Errorf("line %d: expected content %q, got %q", i, tc.content, parts[1])
		}
	}
}

func TestPreviewLines_SingleLine(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "only\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := previewLines(path, hashes[0], hashes[0]); err != nil {
			t.Fatalf("previewLines: %v", err)
		}
	})

	want := hashes[0] + " only\n"
	if out != want {
		t.Errorf("want %q, got %q", want, out)
	}
}

func TestPreviewLines_FullFileMatchesRead(t *testing.T) {
	// Previewing the entire file should produce identical output to rh read.
	setupCache(t)
	path := tempFile(t, "line0\nline1\nline2\n")
	hashes := readHashesFor(t, path)

	readOut := captureStdout(t, func() { readFile(path) })
	previewOut := captureStdout(t, func() {
		previewLines(path, hashes[0], hashes[2])
	})

	if readOut != previewOut {
		t.Errorf("preview of full file differs from read output\nread:    %q\npreview: %q", readOut, previewOut)
	}
}

func TestPreviewLines_DoesNotModifyFile(t *testing.T) {
	setupCache(t)
	original := "line0\nline1\nline2\n"
	path := tempFile(t, original)
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		previewLines(path, hashes[0], hashes[2])
	})

	got, _ := os.ReadFile(path)
	if string(got) != original {
		t.Errorf("previewLines modified the file: want %q, got %q", original, string(got))
	}
}

func TestPreviewLines_ExcludesLinesOutsideRange(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "outside\ntarget\noutside\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := previewLines(path, hashes[1], hashes[1]); err != nil {
			t.Fatalf("previewLines: %v", err)
		}
	})

	if strings.Contains(out, "outside") {
		t.Errorf("expected lines outside range to be excluded, got:\n%s", out)
	}
	if !strings.Contains(out, "target") {
		t.Errorf("expected 'target' in output, got:\n%s", out)
	}
}

func TestPreviewLines_StartAfterEnd(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "a\nb\nc\n")
	hashes := readHashesFor(t, path)

	err := previewLines(path, hashes[2], hashes[0])
	if err == nil || !strings.Contains(err.Error(), "start hash") {
		t.Errorf("expected 'start hash' error, got %v", err)
	}
}

func TestPreviewLines_InvalidStartHash(t *testing.T) {
	err := previewLines("f", "bad!", indexToHash(0))
	if err == nil || !strings.Contains(err.Error(), "start hash") {
		t.Errorf("expected invalid start hash error, got %v", err)
	}
}

func TestPreviewLines_InvalidEndHash(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line\n")
	hashes := readHashesFor(t, path)

	err := previewLines(path, hashes[0], "bad!")
	if err == nil || !strings.Contains(err.Error(), "end hash") {
		t.Errorf("expected invalid end hash error, got %v", err)
	}
}

func TestPreviewLines_FileNotFound(t *testing.T) {
	// No cache entry for this file — findHashIndex will fail, returning an error.
	err := previewLines("/nonexistent/rh_missing.txt", indexToHash(0), indexToHash(1))
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestPreviewLines_OutputFormat(t *testing.T) {
	// Every output line must be exactly "hash<space>content".
	setupCache(t)
	path := tempFile(t, "foo\nbar\nbaz\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := previewLines(path, hashes[0], hashes[2]); err != nil {
			t.Fatalf("previewLines: %v", err)
		}
	})

	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			t.Errorf("line %d not in 'hash content' format: %q", i, line)
			continue
		}
		if len(parts[0]) != 4 {
			t.Errorf("line %d hash is not 4 chars: %q", i, parts[0])
		}
		for _, c := range parts[0] {
			if c < 'a' || c > 'z' {
				t.Errorf("line %d hash %q contains non-lowercase char %c", i, parts[0], c)
			}
		}
	}
}
