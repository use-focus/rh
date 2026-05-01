package main

import (
	"os"
	"strings"
	"testing"
)

func TestAppendToFile_Basic(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "existing\n")

	captureStdout(t, func() {
		if err := appendToFile(path, "new line\n"); err != nil {
			t.Fatalf("appendToFile: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "existing\nnew line\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppendToFile_NoTrailingNewline(t *testing.T) {
	// File has no trailing newline — a separator newline must be inserted first.
	setupCache(t)
	path := tempFile(t, "no-newline")

	captureStdout(t, func() {
		if err := appendToFile(path, "appended\n"); err != nil {
			t.Fatalf("appendToFile: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "no-newline\nappended\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppendToFile_EmptyFile(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "")

	captureStdout(t, func() {
		if err := appendToFile(path, "first\n"); err != nil {
			t.Fatalf("appendToFile: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "first\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppendToFile_MultiLine(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\n")

	captureStdout(t, func() {
		if err := appendToFile(path, "line2\nline3\nline4\n"); err != nil {
			t.Fatalf("appendToFile: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "line1\nline2\nline3\nline4\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppendToFile_PreservesExistingContent(t *testing.T) {
	setupCache(t)
	original := "do not touch\nme either\n"
	path := tempFile(t, original)

	captureStdout(t, func() { appendToFile(path, "extra\n") })

	got, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(got), original) {
		t.Errorf("existing content was modified: got %q", string(got))
	}
}

func TestAppendToFile_CreatesFile(t *testing.T) {
	// File does not yet exist — should be created.
	setupCache(t)
	path := t.TempDir() + "/newfile.txt"

	captureStdout(t, func() {
		if err := appendToFile(path, "hello\n"); err != nil {
			t.Fatalf("appendToFile on new file: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "hello\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppendToFile_SequentialAppends(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\n")

	captureStdout(t, func() { appendToFile(path, "line2\n") })
	captureStdout(t, func() { appendToFile(path, "line3\n") })

	got, _ := os.ReadFile(path)
	want := "line1\nline2\nline3\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppendToFile_DiffShowsNewWriteBlock(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "existing\n")

	out := captureStdout(t, func() {
		if err := appendToFile(path, "appended\n"); err != nil {
			t.Fatalf("appendToFile: %v", err)
		}
	})

	if !strings.Contains(out, "<NewLines>") {
		t.Errorf("expected '<NewLines>' block in output, got:\n%s", out)
	}
	if !strings.Contains(out, "appended") {
		t.Errorf("expected 'appended' in <NewLines> block, got:\n%s", out)
	}
}

func TestAppendToFile_DiffHasNoRemovedLines(t *testing.T) {
	// Append never removes lines — no "- " lines should appear.
	setupCache(t)
	path := tempFile(t, "existing\n")

	out := captureStdout(t, func() {
		appendToFile(path, "new\n")
	})

	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "- ") {
			t.Errorf("unexpected removed line in append output: %q\nfull output:\n%s", line, out)
		}
	}
}

func TestAppendToFile_DiffShowsContext(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "ctx1\nctx2\nctx3\n")

	out := captureStdout(t, func() {
		appendToFile(path, "new\n")
	})

	// The last two lines of the existing file should appear as context.
	for _, ctx := range []string{"ctx2", "ctx3"} {
		if !strings.Contains(out, ctx) {
			t.Errorf("expected context line %q in output, got:\n%s", ctx, out)
		}
	}
}

func TestAppendToFile_DiffMultiLineAppend(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "existing\n")

	out := captureStdout(t, func() {
		appendToFile(path, "lineA\nlineB\nlineC\n")
	})

	if !strings.Contains(out, "<NewLines>") {
		t.Errorf("expected '<NewLines>' block in output, got:\n%s", out)
	}
	for _, added := range []string{"lineA", "lineB", "lineC"} {
		if !strings.Contains(out, added) {
			t.Errorf("expected %q in output, got:\n%s", added, out)
		}
	}
}

func TestAppendToFile_HashesPersistForExistingLines(t *testing.T) {
	// Existing lines must keep their hashes after an append.
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\n")
	before := readHashesFor(t, path)

	captureStdout(t, func() { appendToFile(path, "gamma\n") })

	after := readHashesFor(t, path)

	if after[0] != before[0] {
		t.Errorf("line 0 hash changed after append: was %s, now %s", before[0], after[0])
	}
	if after[1] != before[1] {
		t.Errorf("line 1 hash changed after append: was %s, now %s", before[1], after[1])
	}
}
