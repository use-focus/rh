package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// withStdin replaces os.Stdin with a reader backed by s for the duration of fn.
func withStdin(t *testing.T, s string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if _, err := io.WriteString(w, s); err != nil {
		t.Fatalf("writing to stdin pipe: %v", err)
	}
	w.Close()

	old := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = old
		r.Close()
	}()

	fn()
}

func TestWrite_StdinHeredoc(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\nline2\nline3\n")
	hashes := readHashesFor(t, path)

	withStdin(t, "replaced\n", func() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		captureStdout(t, func() {
			if err := replaceLines(path, hashes[1], hashes[1], string(content)); err != nil {
				t.Fatalf("replaceLines: %v", err)
			}
		})
	})

	got, _ := os.ReadFile(path)
	want := "line1\nreplaced\nline3\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestWrite_StdinWithQuotesAndSpecialChars(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)

	special := "it's a \"test\" with $vars and `backticks`\n"
	withStdin(t, special, func() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		captureStdout(t, func() {
			if err := replaceLines(path, hashes[0], hashes[0], string(content)); err != nil {
				t.Fatalf("replaceLines: %v", err)
			}
		})
	})

	got, _ := os.ReadFile(path)
	want := "it's a \"test\" with $vars and `backticks`\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestWrite_StdinMultiLine(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "before\ntarget\nafter\n")
	hashes := readHashesFor(t, path)

	withStdin(t, "line A\nline B\nline C\n", func() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		captureStdout(t, func() {
			if err := replaceLines(path, hashes[1], hashes[1], string(content)); err != nil {
				t.Fatalf("replaceLines: %v", err)
			}
		})
	})

	got, _ := os.ReadFile(path)
	want := "before\nline A\nline B\nline C\nafter\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppend_StdinPipe(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "existing\n")

	withStdin(t, "appended\n", func() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		captureStdout(t, func() {
			if err := appendToFile(path, string(content)); err != nil {
				t.Fatalf("appendToFile: %v", err)
			}
		})
	})

	got, _ := os.ReadFile(path)
	want := "existing\nappended\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestAppend_StdinMultiLine(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\n")

	withStdin(t, "line2\nline3\n", func() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		captureStdout(t, func() {
			appendToFile(path, string(content))
		})
	})

	got, _ := os.ReadFile(path)
	want := "line1\nline2\nline3\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestWrite_StdinEmptyContent(t *testing.T) {
	// Empty stdin deletes the target lines.
	setupCache(t)
	path := tempFile(t, "keep\ndelete me\nalso keep\n")
	hashes := readHashesFor(t, path)

	withStdin(t, "", func() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		captureStdout(t, func() {
			replaceLines(path, hashes[1], hashes[1], string(content))
		})
	})

	got, _ := os.ReadFile(path)
	want := "keep\nalso keep\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestWrite_StdinDiffPrinted(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\nline2\nline3\n")
	hashes := readHashesFor(t, path)

	var out string
	withStdin(t, "new\n", func() {
		content, _ := io.ReadAll(os.Stdin)
		out = captureStdout(t, func() {
			replaceLines(path, hashes[1], hashes[1], string(content))
		})
	})

	if !strings.Contains(out, "<NewLines>") {
		t.Errorf("expected '<NewLines>' block in output, got:\n%s", out)
	}
	if !strings.Contains(out, "new") {
		t.Errorf("expected 'new' in output, got:\n%s", out)
	}
}
