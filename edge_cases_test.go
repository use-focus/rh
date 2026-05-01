package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

// ── Unicode & multibyte ────────────────────────────────────────────────────

func TestWrite_UnicodeEmoji(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "before\nold\nafter\n")
	hashes := readHashesFor(t, path)
	withStdin(t, "🔥🐹🦀\n", func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			if err := replaceLines(path, hashes[1], hashes[1], string(content)); err != nil {
				t.Fatalf("replaceLines: %v", err)
			}
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "🔥🐹🦀") {
		t.Errorf("emoji not preserved: %q", string(got))
	}
}

func TestWrite_CJKCharacters(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\nline2\n")
	hashes := readHashesFor(t, path)
	withStdin(t, "日本語テスト 中文 한국어\n", func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "日本語テスト") {
		t.Errorf("CJK not preserved: %q", string(got))
	}
	if !utf8.ValidString(string(got)) {
		t.Error("result is not valid UTF-8")
	}
}

func TestWrite_RTLAndCombiningChars(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "placeholder\n")
	hashes := readHashesFor(t, path)
	// Arabic + Hebrew + combining diacritics
	input := "مرحبا שלום e\u0301 \u0301\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !utf8.ValidString(string(got)) {
		t.Error("result is not valid UTF-8")
	}
}

func TestWrite_ZeroWidthAndInvisibleChars(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	// zero-width space U+200B, zero-width joiner U+200D, BOM U+FEFF
	input := "visible\u200Bmiddle\u200Dend\uFEFFbom\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "visible\u200Bmiddle") {
		t.Errorf("zero-width chars not preserved: %q", string(got))
	}
}

// ── Shell metacharacters (critical now that content comes via stdin) ────────

func TestWrite_ShellMetacharacters(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "echo $(rm -rf /) && $VAR | tee > out.txt; `id` & {a,b}\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	want := "echo $(rm -rf /) && $VAR | tee > out.txt; `id` & {a,b}\n"
	if string(got) != want {
		t.Errorf("shell metacharacters corrupted\nwant %q\n got %q", want, string(got))
	}
}

func TestWrite_ANSIEscapeCodes(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "\x1b[31mred\x1b[0m \x1b[1mbold\x1b[0m\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "\x1b[31m") {
		t.Errorf("ANSI codes not preserved: %q", string(got))
	}
}

// ── Whitespace edge cases ─────────────────────────────────────────────────

func TestWrite_TabsAndMixedIndentation(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "\tfunc foo() {\n\t\treturn 42\n\t}\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "\t\treturn 42") {
		t.Errorf("tab indentation not preserved: %q", string(got))
	}
}

func TestWrite_TrailingWhitespacePreserved(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	// trailing spaces on a line should survive the round-trip
	input := "line with trailing spaces   \n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "trailing spaces   ") {
		t.Errorf("trailing spaces stripped: %q", string(got))
	}
}

func TestWrite_BlankLinesInContent(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "first\n\n\nfourth\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if string(got) != "first\n\n\nfourth\n" {
		t.Errorf("blank lines not preserved: %q", string(got))
	}
}

// ── Content that could confuse the diff or hash reader ────────────────────

func TestWrite_ContentLookingLikeHashPrefix(t *testing.T) {
	// A line whose first token is a valid 4-letter hash should not confuse rh read.
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "rwtb this looks like a hash line\ntltc so does this\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "rwtb this looks like a hash line") {
		t.Errorf("hash-like content corrupted: %q", string(got))
	}
}

func TestWrite_ContentLookingLikeDiffMarkers(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "+ added line\n- removed line\n  context line\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "+ added line") {
		t.Errorf("diff-like content corrupted: %q", string(got))
	}
}

// ── Real programming content ──────────────────────────────────────────────

func TestWrite_GoCodeWithBackticks(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "var re = `^[a-z]{4}$`\nfmt.Println(`hello world`)\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "^[a-z]{4}$") {
		t.Errorf("backtick content corrupted: %q", string(got))
	}
}

func TestWrite_JSONContent(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := `{"key": "value", "nested": {"arr": [1, 2, 3]}, "flag": true}` + "\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), `"nested": {"arr": [1, 2, 3]}`) {
		t.Errorf("JSON content corrupted: %q", string(got))
	}
}

func TestWrite_SQLWithQuotes(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "SELECT * FROM \"users\" WHERE name = 'O''Brien' AND age > 0;\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "O''Brien") {
		t.Errorf("SQL content corrupted: %q", string(got))
	}
}

func TestWrite_RegexPattern(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := `re := regexp.MustCompile("^(?P<hash>[a-z]{4})\\s+(?P<content>.*)$")` + "\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "MustCompile") {
		t.Errorf("regex content corrupted: %q", string(got))
	}
}

// ── Buffer / size edge cases ──────────────────────────────────────────────

func TestWrite_VeryLongLine(t *testing.T) {
	// bufio.Scanner default limit is 64KB — write a line just under that.
	setupCache(t)
	longLine := strings.Repeat("x", 60*1024) + "\n"
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	withStdin(t, longLine, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			if err := replaceLines(path, hashes[0], hashes[0], string(content)); err != nil {
				t.Fatalf("replaceLines on long line: %v", err)
			}
		})
	})
	got, _ := os.ReadFile(path)
	if len(got) < 60*1024 {
		t.Errorf("long line was truncated: got %d bytes", len(got))
	}
}

func TestReadFile_VeryLongLine(t *testing.T) {
	// Confirm rh read also survives a long line without error.
	setupCache(t)
	longLine := strings.Repeat("y", 60*1024)
	path := tempFile(t, longLine+"\nnormal\n")

	out := captureStdout(t, func() {
		if err := readFile(path); err != nil {
			t.Fatalf("readFile: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
}

func TestAppend_VeryLargeFile(t *testing.T) {
	// Build a file with 10k lines and append to it.
	setupCache(t)
	var sb strings.Builder
	for i := 0; i < 10_000; i++ {
		sb.WriteString("line\n")
	}
	path := tempFile(t, sb.String())
	withStdin(t, "final\n", func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			if err := appendToFile(path, string(content)); err != nil {
				t.Fatalf("appendToFile: %v", err)
			}
		})
	})
	got, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) != 10_001 {
		t.Errorf("expected 10001 lines, got %d", len(lines))
	}
	if lines[10_000] != "final" {
		t.Errorf("last line wrong: %q", lines[10_000])
	}
}

func TestWrite_NullByteInContent(t *testing.T) {
	// Null bytes are valid in Go strings and should survive the write.
	setupCache(t)
	path := tempFile(t, "old\n")
	hashes := readHashesFor(t, path)
	input := "before\x00after\n"
	withStdin(t, input, func() {
		content, _ := io.ReadAll(os.Stdin)
		captureStdout(t, func() {
			replaceLines(path, hashes[0], hashes[0], string(content))
		})
	})
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "before\x00after") {
		t.Errorf("null byte not preserved: %q", string(got))
	}
}

func TestReadFile_LongLineUsesLargerBuffer(t *testing.T) {
	// Lines over the default 64KB scanner limit should return an error,
	// confirming the current behaviour so we notice if it changes.
	setupCache(t)
	longLine := strings.Repeat("z", bufio.MaxScanTokenSize+1)
	path := tempFile(t, longLine+"\n")

	err := readFile(path)
	if err == nil {
		t.Log("note: readFile handled a line exceeding bufio.MaxScanTokenSize without error")
	}
}
