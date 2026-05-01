package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// readFileLines reads all lines from a file using a 1MB scanner buffer.
func readFileLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func readFile(path string) error {
	lines, err := readFileLines(path)
	if err != nil {
		return err
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	c, err := loadCache()
	if err != nil {
		return err
	}

	state := c.Files[abs]

	// If the hash list doesn't match the file, rebuild it from scratch.
	if len(state.LineHashes) != len(lines) {
		state.LineHashes = buildHashList(c, len(lines))
	}

	for i, line := range lines {
		fmt.Printf("%s %s\n", state.LineHashes[i], line)
	}

	state.LastReadAt = time.Now()
	state.ContentHash, err = fileContentHash(abs)
	if err != nil {
		return err
	}
	c.Files[abs] = state
	return saveCache(c)
}

func replaceLines(path, startHash, endHash, newContentStr string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	c, err := loadCache()
	if err != nil {
		return err
	}

	state := c.Files[abs]

	startIdx, err := findHashIndex(state.LineHashes, startHash)
	if err != nil {
		return fmt.Errorf("invalid start hash: %v", err)
	}
	endIdx, err := findHashIndex(state.LineHashes, endHash)
	if err != nil {
		return fmt.Errorf("invalid end hash: %v", err)
	}
	if startIdx > endIdx {
		return fmt.Errorf("start hash (%s) represents a line after end hash (%s)", startHash, endHash)
	}

	lines, err := readFileLines(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	newLines := splitContent(newContentStr)
	newHashes := buildHashList(c, len(newLines))

	// Build new file content.
	var result []string
	result = append(result, lines[:startIdx]...)
	result = append(result, newLines...)
	if endIdx+1 < len(lines) {
		result = append(result, lines[endIdx+1:]...)
	}

	// Build new hash list — keep hashes outside the replaced range, assign fresh ones inside.
	var newHashList []string
	newHashList = append(newHashList, state.LineHashes[:startIdx]...)
	newHashList = append(newHashList, newHashes...)
	if endIdx+1 < len(state.LineHashes) {
		newHashList = append(newHashList, state.LineHashes[endIdx+1:]...)
	}

	// Write file.
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if err := writeLines(outFile, result); err != nil {
		outFile.Close()
		return err
	}
	if err := outFile.Close(); err != nil {
		return err
	}

	printWriteResult(lines, state.LineHashes, startIdx, endIdx, newLines, newHashes)

	// Update cache — new hash list, bump LastReadAt so the mtime guard doesn't fire.
	state.LineHashes = newHashList
	state.LastReadAt = time.Now()
	state.ContentHash, err = fileContentHash(abs)
	if err != nil {
		return err
	}
	c.Files[abs] = state
	return saveCache(c)
}

func previewLines(path, startHash, endHash string) error {
	// Validate hash formats before any I/O so format errors surface immediately.
	if _, err := findHashIndex(nil, startHash); err != nil {
		if isFormatError(startHash) {
			return fmt.Errorf("invalid start hash: %v", err)
		}
	}
	if _, err := findHashIndex(nil, endHash); err != nil {
		if isFormatError(endHash) {
			return fmt.Errorf("invalid end hash: %v", err)
		}
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Read the full file and build/reuse the complete hash list — same as rh read.
	// This registers the file so a write can follow without re-reading.
	lines, err := readFileLines(path)
	if err != nil {
		return err
	}

	c, err := loadCache()
	if err != nil {
		return err
	}

	state := c.Files[abs]

	if len(state.LineHashes) != len(lines) {
		state.LineHashes = buildHashList(c, len(lines))
	}

	startIdx, err := findHashIndex(state.LineHashes, startHash)
	if err != nil {
		return fmt.Errorf("invalid start hash: %v", err)
	}
	endIdx, err := findHashIndex(state.LineHashes, endHash)
	if err != nil {
		return fmt.Errorf("invalid end hash: %v", err)
	}
	if startIdx > endIdx {
		return fmt.Errorf("start hash (%s) represents a line after end hash (%s)", startHash, endHash)
	}

	for i := startIdx; i <= endIdx && i < len(lines); i++ {
		fmt.Printf("%s %s\n", state.LineHashes[i], lines[i])
	}

	state.LastReadAt = time.Now()
	state.ContentHash, err = fileContentHash(abs)
	if err != nil {
		return err
	}
	c.Files[abs] = state
	return saveCache(c)
}

type grepFileState struct {
	hashes []string
}

func grepFiles(grepArgs []string) error {
	if len(grepArgs) == 0 {
		return fmt.Errorf("missing grep pattern")
	}

	c, err := loadCache()
	if err != nil {
		return err
	}

	records, err := grepRecords(grepArgs)
	if err != nil {
		return err
	}

	files := make(map[string]grepFileState)
	renderer := grepBlockRenderer{}
	for _, record := range records {
		if record.isSeparator() {
			renderer.endGroup()
			continue
		}
		hash, ok, err := hashForGrepRecord(c, files, record.path, record.lineNumber)
		if err != nil {
			return err
		}
		if !ok {
			renderer.endGroup()
			fmt.Println(record.displayRaw())
			continue
		}
		renderer.print(record.path, hash, record.content)
	}

	return saveCache(c)
}

type grepBlockRenderer struct {
	currentPath string
	inGroup     bool
	wroteGroup  bool
}

func (r *grepBlockRenderer) print(path, hash, content string) {
	if !r.inGroup || r.currentPath != path {
		r.startGroup(path)
	}
	fmt.Printf("%s %s\n", hash, content)
}

func (r *grepBlockRenderer) startGroup(path string) {
	if r.wroteGroup {
		fmt.Println()
	}
	fmt.Printf("MATCH - %s:\n", path)
	r.currentPath = path
	r.inGroup = true
	r.wroteGroup = true
}

func (r *grepBlockRenderer) endGroup() {
	r.inGroup = false
	r.currentPath = ""
}

func hashForGrepRecord(c *Cache, files map[string]grepFileState, path string, lineNumber int) (string, bool, error) {
	if lineNumber < 1 {
		return "", false, nil
	}

	file, ok := files[path]
	if !ok {
		lines, err := readFileLines(path)
		if err != nil {
			return "", false, err
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return "", false, err
		}

		state := c.Files[abs]
		if len(state.LineHashes) != len(lines) {
			state.LineHashes = buildHashList(c, len(lines))
		}
		state.LastReadAt = time.Now()
		state.ContentHash, err = fileContentHash(abs)
		if err != nil {
			return "", false, err
		}
		c.Files[abs] = state

		file = grepFileState{
			hashes: state.LineHashes,
		}
		files[path] = file
	}

	i := lineNumber - 1
	if i < 0 || i >= len(file.hashes) {
		return "", false, nil
	}
	return file.hashes[i], true, nil
}

func grepRecords(grepArgs []string) ([]grepRecord, error) {
	args := []string{"-H", "-n", "--null"}
	if shouldUseExtendedGrep(grepArgs) {
		args = append(args, "-E")
	}
	args = append(args, forceFilenameOutput(grepArgs)...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("grep", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("grep failed: %s", msg)
	}

	return parseGrepRecords(stdout.String()), nil
}

func forceFilenameOutput(grepArgs []string) []string {
	forced := make([]string, 0, len(grepArgs))
	for _, arg := range grepArgs {
		switch {
		case arg == "-h" || arg == "--no-filename":
			continue
		case strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.Contains(arg, "h"):
			withoutH := strings.ReplaceAll(arg, "h", "")
			if withoutH != "-" {
				forced = append(forced, withoutH)
			}
		default:
			forced = append(forced, arg)
		}
	}
	return forced
}

func shouldUseExtendedGrep(grepArgs []string) bool {
	for _, arg := range grepArgs {
		if strings.Contains(arg, `\|`) {
			return false
		}
		if arg == "--" {
			return true
		}
		if arg == "-E" || arg == "-F" || arg == "-G" || arg == "-P" ||
			arg == "--extended-regexp" || arg == "--fixed-strings" ||
			arg == "--basic-regexp" || arg == "--perl-regexp" {
			return false
		}
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			for _, ch := range arg[1:] {
				if ch == 'E' || ch == 'F' || ch == 'G' || ch == 'P' {
					return false
				}
			}
		}
	}
	return true
}

type grepRecord struct {
	raw           string
	path          string
	lineNumber    int
	fileSeparator byte
	lineSeparator byte
	content       string
}

func (r grepRecord) displayRaw() string {
	return strings.ReplaceAll(r.raw, "\x00", ":")
}

func (r grepRecord) isSeparator() bool {
	return r.raw == "--"
}

func parseGrepRecords(grepOutput string) []grepRecord {
	var records []grepRecord
	for _, line := range strings.Split(strings.TrimRight(grepOutput, "\n"), "\n") {
		record := grepRecord{raw: line}
		null := strings.IndexByte(line, 0)
		if null == -1 {
			records = append(records, record)
			continue
		}

		record.path = line[:null]
		record.fileSeparator = ':'
		rest := line[null+1:]
		sep := strings.IndexAny(rest, ":-")
		if sep == -1 {
			records = append(records, record)
			continue
		}

		lineNumber, err := strconv.Atoi(rest[:sep])
		if err != nil || lineNumber < 1 {
			records = append(records, record)
			continue
		}

		record.lineNumber = lineNumber
		record.lineSeparator = rest[sep]
		record.content = rest[sep+1:]
		records = append(records, record)
	}
	return records
}

func appendToFile(path, content string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	existingData, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var lines []string
	if len(existingData) > 0 {
		raw := strings.TrimSuffix(string(existingData), "\n")
		lines = strings.Split(raw, "\n")
	}

	c, err := loadCache()
	if err != nil {
		return err
	}

	state := c.Files[abs]
	newLines := splitContent(content)
	newHashes := buildHashList(c, len(newLines))

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(f)
	if len(existingData) > 0 && !strings.HasSuffix(string(existingData), "\n") {
		if _, err := writer.WriteString("\n"); err != nil {
			f.Close()
			return err
		}
	}
	for _, line := range newLines {
		if _, err := writer.WriteString(strings.TrimRight(line, "\r") + "\n"); err != nil {
			f.Close()
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// startIdx is after the last existing line; endIdx is before it (nothing replaced).
	printWriteResult(lines, state.LineHashes, len(lines), len(lines)-1, newLines, newHashes)

	state.LineHashes = append(state.LineHashes, newHashes...)
	state.LastReadAt = time.Now()
	state.ContentHash, err = fileContentHash(abs)
	if err != nil {
		return err
	}
	c.Files[abs] = state
	return saveCache(c)
}

// splitContent normalises a raw content string into lines,
// stripping one trailing newline and trimming \r from each line.
func splitContent(s string) []string {
	if len(s) == 0 {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	if len(s) == 0 {
		return []string{""}
	}
	parts := strings.Split(s, "\n")
	for i, p := range parts {
		parts[i] = strings.TrimRight(p, "\r")
	}
	return parts
}

// writeLines writes each line followed by a newline to w.
func writeLines(w *os.File, lines []string) error {
	writer := bufio.NewWriter(w)
	for _, line := range lines {
		if _, err := writer.WriteString(strings.TrimRight(line, "\r") + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}
