package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepFile_RegexpAlternation(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "const user = useQuery()\nconst save = useMutation()\nconst idle = noop()\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{`useQuery|useMutation`, path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, "useQuery") {
		t.Errorf("expected useQuery match, got:\n%s", out)
	}
	if !strings.Contains(out, "useMutation") {
		t.Errorf("expected useMutation match, got:\n%s", out)
	}
	if strings.Contains(out, "noop") {
		t.Errorf("unexpected non-matching line, got:\n%s", out)
	}
	if !strings.Contains(out, "MATCH - "+path+":") {
		t.Errorf("expected match block header, got:\n%s", out)
	}
}

func TestGrepFile_GrepStyleEscapedAlternation(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "CurrentUser\nRequireSuperadmin\nRegularUser\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{`CurrentUser\|RequireSuperadmin`, path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, "CurrentUser") {
		t.Errorf("expected CurrentUser match, got:\n%s", out)
	}
	if !strings.Contains(out, "RequireSuperadmin") {
		t.Errorf("expected RequireSuperadmin match, got:\n%s", out)
	}
	if strings.Contains(out, "RegularUser") {
		t.Errorf("unexpected non-matching line, got:\n%s", out)
	}
}

func TestGrepFile_InvalidRegexp(t *testing.T) {
	path := tempFile(t, "anything\n")
	err := grepFiles([]string{`[`, path})
	if err == nil || !strings.Contains(err.Error(), "grep failed") {
		t.Fatalf("expected grep failure, got %v", err)
	}
}

func TestGrepFile_PassesGrepFlags(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "Needle\nneedle\nhaystack\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"-i", "needle", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, "Needle") {
		t.Errorf("expected case-insensitive match for Needle, got:\n%s", out)
	}
	if !strings.Contains(out, "needle") {
		t.Errorf("expected case-insensitive match for needle, got:\n%s", out)
	}
	if strings.Contains(out, "haystack") {
		t.Errorf("unexpected non-matching line, got:\n%s", out)
	}
}

func TestGrepFile_ContextFlagsIncludeContextLines(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "before\nneedle\nafter\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"-A", "1", "needle", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, "needle") {
		t.Errorf("expected matching line, got:\n%s", out)
	}
	if !strings.Contains(out, "after") {
		t.Errorf("expected grep context line, got:\n%s", out)
	}
	if strings.Contains(out, "before") {
		t.Errorf("unexpected line outside grep context, got:\n%s", out)
	}
}

func TestGrepFile_MultipleFilesIncludePathAndHash(t *testing.T) {
	setupCache(t)
	first := tempFile(t, "target first\n")
	second := tempFile(t, "target second\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"target", first, second}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, "MATCH - "+first+":") {
		t.Errorf("expected first file path in output, got:\n%s", out)
	}
	if !strings.Contains(out, "MATCH - "+second+":") {
		t.Errorf("expected second file path in output, got:\n%s", out)
	}
	if !strings.Contains(out, " target first") {
		t.Errorf("expected first file content in output, got:\n%s", out)
	}
	if !strings.Contains(out, " target second") {
		t.Errorf("expected second file content in output, got:\n%s", out)
	}
}

func TestGrepFile_OnlyMatchingOutputKeepsGrepContent(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "prefix needle suffix\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"-o", "needle", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, " needle") {
		t.Errorf("expected grep -o content, got:\n%s", out)
	}
	if strings.Contains(out, "prefix") || strings.Contains(out, "suffix") {
		t.Errorf("expected only matched content from grep -o, got:\n%s", out)
	}
}

func TestGrepFile_ForcesFilenameWhenNoFilenameFlagIsPassed(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "needle\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"-h", "needle", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, "MATCH - "+path+":") {
		t.Errorf("expected file path despite -h, got:\n%s", out)
	}
	if !strings.Contains(out, " needle") {
		t.Errorf("expected matching content, got:\n%s", out)
	}
}

func TestGrepFiles_OutputReplacesLineNumbersWithExpectedHashes(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "first\nsecond target\nthird target\n")
	wantHashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"target", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 2 output lines, got %d:\n%s", len(lines), out)
	}

	want := []string{
		"MATCH - " + path + ":",
		wantHashes[1] + " second target",
		wantHashes[2] + " third target",
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d\nwant %q\n got %q", i, want[i], lines[i])
		}
	}
}

func TestGrepFiles_ContextOutputUsesHashAndContextSeparator(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "before\nneedle\nafter\n")
	wantHashes := readHashesFor(t, path)

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"-A", "1", "needle", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 2 output lines, got %d:\n%s", len(lines), out)
	}
	wantHeader := "MATCH - " + path + ":"
	wantMatch := wantHashes[1] + " needle"
	wantContext := wantHashes[2] + " after"
	if lines[0] != wantHeader {
		t.Errorf("header line\nwant %q\n got %q", wantHeader, lines[0])
	}
	if lines[1] != wantMatch {
		t.Errorf("match line\nwant %q\n got %q", wantMatch, lines[1])
	}
	if lines[2] != wantContext {
		t.Errorf("context line\nwant %q\n got %q", wantContext, lines[2])
	}
}

func TestGrepFiles_ContextGroupsKeepGrepSeparator(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "target one\nafter one\ngap\ntarget two\nafter two\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"-A", "1", "target", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 7 {
		t.Fatalf("expected 5 output lines, got %d:\n%s", len(lines), out)
	}
	if lines[3] != "" {
		t.Fatalf("expected blank line between match groups, got %q\n%s", lines[3], out)
	}
	if lines[0] != "MATCH - "+path+":" || lines[4] != "MATCH - "+path+":" {
		t.Fatalf("expected a match header for each separated group, got:\n%s", out)
	}
}

func TestGrepFiles_NoMatchesProducesNoOutputAndNoCacheEntry(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\n")

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"needle", path}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})
	if out != "" {
		t.Fatalf("expected no output, got:\n%s", out)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	c, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if _, ok := c.Files[abs]; ok {
		t.Fatalf("expected unmatched file not to be registered")
	}
}

func TestGrepFiles_RegistersEveryMatchedFile(t *testing.T) {
	setupCache(t)
	first := tempFile(t, "target first\n")
	second := tempFile(t, "target second\n")

	captureStdout(t, func() {
		if err := grepFiles([]string{"target", first, second}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	c, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	for _, path := range []string{first, second} {
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("Abs: %v", err)
		}
		state, ok := c.Files[abs]
		if !ok {
			t.Fatalf("expected %s to be registered", path)
		}
		if len(state.LineHashes) != 1 {
			t.Fatalf("expected one hash for %s, got %v", path, state.LineHashes)
		}
	}
}

func TestGrepFiles_RecursiveSearchHashesDiscoveredFiles(t *testing.T) {
	setupCache(t)
	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "nested", "second.txt")
	if err := os.MkdirAll(filepath.Dir(second), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(first, []byte("target one\n"), 0644); err != nil {
		t.Fatalf("WriteFile first: %v", err)
	}
	if err := os.WriteFile(second, []byte("target two\n"), 0644); err != nil {
		t.Fatalf("WriteFile second: %v", err)
	}

	out := captureStdout(t, func() {
		if err := grepFiles([]string{"-R", "target", dir}); err != nil {
			t.Fatalf("grepFiles: %v", err)
		}
	})

	if !strings.Contains(out, "MATCH - "+first+":") {
		t.Errorf("expected recursive output for first file, got:\n%s", out)
	}
	if !strings.Contains(out, "MATCH - "+second+":") {
		t.Errorf("expected recursive output for nested file, got:\n%s", out)
	}
	if strings.Contains(out, ":1:") || strings.Contains(out, ":2:") {
		t.Errorf("expected hashes instead of line numbers, got:\n%s", out)
	}
}

func TestGrepRecords_ParsesColonInPath(t *testing.T) {
	records := parseGrepRecords("/tmp/rh:colon.txt\x001:needle\n")
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].path != "/tmp/rh:colon.txt" {
		t.Errorf("path: want colon path, got %q", records[0].path)
	}
	if records[0].lineNumber != 1 {
		t.Errorf("lineNumber: want 1, got %d", records[0].lineNumber)
	}
	if records[0].content != "needle" {
		t.Errorf("content: want needle, got %q", records[0].content)
	}
}

func TestForceFilenameOutputRemovesNoFilenameForms(t *testing.T) {
	got := forceFilenameOutput([]string{"-hi", "--no-filename", "needle", "file.txt"})
	want := []string{"-i", "needle", "file.txt"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg %d: want %q, got %q", i, want[i], got[i])
		}
	}
}
