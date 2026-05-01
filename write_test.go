package main

import (
	"os"
	"strings"
	"testing"
)

func TestReplaceLines_Basic(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line 1\nline 2\nline 3\nline 4\nline 5\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[1], hashes[3], "new line A\nnew line B\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "line 1\nnew line A\nnew line B\nline 5\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_FirstLine(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\ngamma\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[0], hashes[0], "replaced\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "replaced\nbeta\ngamma\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_LastLine(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\ngamma\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[2], hashes[2], "replaced\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "alpha\nbeta\nreplaced\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_AllLines(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\ngamma\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[0], hashes[2], "x\ny\nz\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "x\ny\nz\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_ExpandContent(t *testing.T) {
	// Replace 1 line with 3 lines.
	setupCache(t)
	path := tempFile(t, "before\ntarget\nafter\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[1], hashes[1], "a\nb\nc\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "before\na\nb\nc\nafter\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_CollapseContent(t *testing.T) {
	// Replace 3 lines with 1 line.
	setupCache(t)
	path := tempFile(t, "before\nA\nB\nC\nafter\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[1], hashes[3], "collapsed\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "before\ncollapsed\nafter\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_DeleteLines(t *testing.T) {
	// Passing empty content removes the targeted lines entirely.
	setupCache(t)
	path := tempFile(t, "keep\ndelete me\nalso keep\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[1], hashes[1], ""); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "keep\nalso keep\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_CarriageReturnStripped(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\nline2\n")
	hashes := readHashesFor(t, path)

	captureStdout(t, func() {
		if err := replaceLines(path, hashes[0], hashes[0], "windows\r\nstyle\r\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	got, _ := os.ReadFile(path)
	want := "windows\nstyle\nline2\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestReplaceLines_OutOfBounds(t *testing.T) {
	// Hashes not in the cache are rejected — you cannot reference lines that don't exist.
	setupCache(t)
	path := tempFile(t, "line 1\nline 2\n")
	readHashesFor(t, path) // populate cache with 2 hashes

	err := replaceLines(path, indexToHash(99), indexToHash(100), "new line\n")
	if err == nil {
		t.Fatal("expected error for hash not in cache, got nil")
	}
	if !strings.Contains(err.Error(), "start hash") {
		t.Errorf("expected 'start hash' in error, got: %v", err)
	}
}

func TestReplaceLines_StartAfterEnd(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "a\nb\nc\n")
	hashes := readHashesFor(t, path)

	err := replaceLines(path, hashes[2], hashes[0], "new line\n")
	if err == nil || !strings.Contains(err.Error(), "start hash") {
		t.Errorf("expected 'start hash' error, got %v", err)
	}
}

func TestReplaceLines_EmptyFile(t *testing.T) {
	// An empty file has no hashes — replaceLines cannot target it, use appendToFile instead.
	setupCache(t)
	path := tempFile(t, "")
	readHashesFor(t, path) // 0 hashes

	err := replaceLines(path, indexToHash(0), indexToHash(0), "first line\n")
	if err == nil {
		t.Fatal("expected error when writing to empty file with no hashes, got nil")
	}
}

func TestReplaceLines_InvalidStartHash(t *testing.T) {
	err := replaceLines("f", "bad!", indexToHash(0), "x\n")
	if err == nil || !strings.Contains(err.Error(), "start hash") {
		t.Errorf("expected invalid start hash error, got %v", err)
	}
}

func TestReplaceLines_InvalidEndHash(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line\n")
	hashes := readHashesFor(t, path)

	err := replaceLines(path, hashes[0], "bad!", "x\n")
	if err == nil || !strings.Contains(err.Error(), "end hash") {
		t.Errorf("expected invalid end hash error, got %v", err)
	}
}

func TestReplaceLines_DiffShowsNewWriteBlock(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\nline2\nline3\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := replaceLines(path, hashes[1], hashes[1], "replaced\n"); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	if !strings.Contains(out, "<NewLines>") {
		t.Errorf("expected '<NewLines>' block in output, got:\n%s", out)
	}
	if !strings.Contains(out, "replaced") {
		t.Errorf("expected 'replaced' inside <NewLines> block, got:\n%s", out)
	}
	// Removed line should not appear in output.
	if strings.Contains(out, "line2") {
		t.Errorf("removed line 'line2' should not appear in output, got:\n%s", out)
	}
}

func TestReplaceLines_DiffContextLines(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "ctx1\nctx2\ntarget\nctx3\nctx4\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		replaceLines(path, hashes[2], hashes[2], "new\n")
	})

	// diffContextLines lines of context above and below should appear in output.
	for _, ctx := range []string{"ctx1", "ctx2", "ctx3", "ctx4"} {
		if !strings.Contains(out, ctx) {
			t.Errorf("expected context line %q in output, got:\n%s", ctx, out)
		}
	}
}

func TestReplaceLines_DiffOnlyShowsChangedRegion(t *testing.T) {
	// Lines far from the replaced region must not appear in the output.
	setupCache(t)
	path := tempFile(t, "far1\nfar2\nfar3\nctx1\nctx2\ntarget\nctx3\nctx4\nfar5\nfar6\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		replaceLines(path, hashes[5], hashes[5], "new\n")
	})

	for _, absent := range []string{"far1", "far2", "far6"} {
		if strings.Contains(out, absent) {
			t.Errorf("line %q should not appear in output, got:\n%s", absent, out)
		}
	}
}

func TestReplaceLines_HashesPersistOutsideEditRange(t *testing.T) {
	// Lines outside the replaced range must keep their original hashes.
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\ngamma\n")
	before := readHashesFor(t, path)

	captureStdout(t, func() {
		replaceLines(path, before[1], before[1], "replaced\n")
	})

	after := readHashesFor(t, path)

	// Lines 0 and 2 were untouched — their hashes must be unchanged.
	if after[0] != before[0] {
		t.Errorf("line 0 hash changed: was %s, now %s", before[0], after[0])
	}
	if after[2] != before[2] {
		t.Errorf("line 2 hash changed: was %s, now %s", before[2], after[2])
	}
	// Line 1 was replaced — its hash must be new.
	if after[1] == before[1] {
		t.Errorf("line 1 hash should have changed after replacement, still %s", after[1])
	}
}

func TestReplaceLines_DiffShowsDeletedLinesBlock(t *testing.T) {
	// Pure deletion (empty content) must show <DeletedLines> with the deleted content.
	setupCache(t)
	path := tempFile(t, "keep\ndelete me\nalso keep\n")
	hashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := replaceLines(path, hashes[1], hashes[1], ""); err != nil {
			t.Fatalf("replaceLines: %v", err)
		}
	})

	if !strings.Contains(out, "<DeletedLines>") {
		t.Errorf("expected '<DeletedLines>' block for pure deletion, got:\n%s", out)
	}
	if !strings.Contains(out, "delete me") {
		t.Errorf("expected deleted content inside <DeletedLines>, got:\n%s", out)
	}
	// Must NOT show <NewLines> for a pure deletion.
	if strings.Contains(out, "<NewLines>") {
		t.Errorf("unexpected '<NewLines>' block in pure deletion output:\n%s", out)
	}
}
