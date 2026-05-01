package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupCache redirects the cache to a temp dir for the duration of the test.
func setupCache(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	overrideCachePath = filepath.Join(dir, "state.json")
	t.Cleanup(func() { overrideCachePath = "" })
}

// cacheStateFor is a test helper that manually writes a FileState into the cache.
func cacheStateFor(t *testing.T, filePath string, state FileState) {
	t.Helper()
	abs, _ := filepath.Abs(filePath)
	c, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	c.Files[abs] = state
	if err := saveCache(c); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
}

// ── loadCache / saveCache ─────────────────────────────────────────────────────

func TestLoadCache_EmptyOnMissing(t *testing.T) {
	setupCache(t)

	c, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache on missing file: %v", err)
	}
	if len(c.Files) != 0 {
		t.Errorf("expected empty Files map, got %d entries", len(c.Files))
	}
	if c.NextID != 0 {
		t.Errorf("expected NextID=0, got %d", c.NextID)
	}
}

func TestSaveAndLoadCache_RoundTrip(t *testing.T) {
	setupCache(t)

	now := time.Now().Truncate(time.Second)
	c := &Cache{
		NextID: 42,
		Files:  make(map[string]FileState),
	}
	c.Files["/some/file.go"] = FileState{
		LastReadAt: now,
		LineHashes: []string{"abcd", "efgh", "ijkl"},
	}

	if err := saveCache(c); err != nil {
		t.Fatalf("saveCache: %v", err)
	}

	loaded, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if loaded.NextID != 42 {
		t.Errorf("NextID: want 42, got %d", loaded.NextID)
	}
	state, ok := loaded.Files["/some/file.go"]
	if !ok {
		t.Fatal("entry missing after round-trip")
	}
	if !state.LastReadAt.Equal(now) {
		t.Errorf("LastReadAt: want %v, got %v", now, state.LastReadAt)
	}
	if len(state.LineHashes) != 3 || state.LineHashes[1] != "efgh" {
		t.Errorf("LineHashes not preserved: %v", state.LineHashes)
	}
	if state.ContentHash != "" {
		t.Errorf("ContentHash should be empty in this legacy-style round trip, got %q", state.ContentHash)
	}
}

// ── allocHash ─────────────────────────────────────────────────────────────────

func TestAllocHash_ReturnsValidHash(t *testing.T) {
	c := &Cache{Files: make(map[string]FileState)}
	h := allocHash(c)

	if len(h) != 4 {
		t.Errorf("expected 4-char hash, got %q (len %d)", h, len(h))
	}
	for _, ch := range h {
		if ch < 'a' || ch > 'z' {
			t.Errorf("hash %q contains non-lowercase character %c", h, ch)
		}
	}
}

func TestAllocHash_AdvancesNextID(t *testing.T) {
	c := &Cache{NextID: 5, Files: make(map[string]FileState)}
	allocHash(c)
	if c.NextID != 6 {
		t.Errorf("expected NextID=6 after one alloc, got %d", c.NextID)
	}
}

func TestAllocHash_ProducesUniqueHashes(t *testing.T) {
	c := &Cache{Files: make(map[string]FileState)}
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		h := allocHash(c)
		if seen[h] {
			t.Errorf("duplicate hash %q at NextID=%d", h, i)
		}
		seen[h] = true
	}
}

func TestAllocHash_WrapsAtModulus(t *testing.T) {
	c := &Cache{NextID: lcgModulus - 1, Files: make(map[string]FileState)}
	allocHash(c)
	if c.NextID != 0 {
		t.Errorf("expected NextID to wrap to 0, got %d", c.NextID)
	}
}

// ── buildHashList ─────────────────────────────────────────────────────────────

func TestBuildHashList_CorrectCount(t *testing.T) {
	c := &Cache{Files: make(map[string]FileState)}
	hashes := buildHashList(c, 5)
	if len(hashes) != 5 {
		t.Errorf("expected 5 hashes, got %d", len(hashes))
	}
}

func TestBuildHashList_AllUnique(t *testing.T) {
	c := &Cache{Files: make(map[string]FileState)}
	hashes := buildHashList(c, 50)
	seen := make(map[string]bool)
	for _, h := range hashes {
		if seen[h] {
			t.Errorf("duplicate hash %q in buildHashList result", h)
		}
		seen[h] = true
	}
}

func TestBuildHashList_AdvancesNextID(t *testing.T) {
	c := &Cache{NextID: 0, Files: make(map[string]FileState)}
	buildHashList(c, 10)
	if c.NextID != 10 {
		t.Errorf("expected NextID=10 after building 10 hashes, got %d", c.NextID)
	}
}

func TestBuildHashList_ZeroCount(t *testing.T) {
	c := &Cache{NextID: 3, Files: make(map[string]FileState)}
	hashes := buildHashList(c, 0)
	if len(hashes) != 0 {
		t.Errorf("expected empty slice for count=0, got %v", hashes)
	}
	if c.NextID != 3 {
		t.Errorf("expected NextID unchanged at 3, got %d", c.NextID)
	}
}

// ── checkWriteAllowed ─────────────────────────────────────────────────────────

func TestCheckWriteAllowed_NoEntry_Allowed(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\n")

	if err := checkWriteAllowed(path); err != nil {
		t.Errorf("expected nil for untracked file, got: %v", err)
	}
}

func TestCheckWriteAllowed_FreshRead_Allowed(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\n")

	// Simulate a recent rh read by setting LastReadAt to now.
	cacheStateFor(t, path, FileState{LastReadAt: time.Now()})

	if err := checkWriteAllowed(path); err != nil {
		t.Errorf("expected nil after fresh read, got: %v", err)
	}
}

func TestCheckWriteAllowed_ExternalModification_Blocked(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "line1\n")

	// Backdate LastReadAt so the file's mtime appears to be after the read.
	cacheStateFor(t, path, FileState{LastReadAt: time.Now().Add(-2 * time.Second)})

	err := checkWriteAllowed(path)
	if err == nil {
		t.Fatal("expected error for external modification, got nil")
	}
	if !strings.Contains(err.Error(), "modified externally") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCheckWriteAllowed_ContentHashAllowsUnchangedFile(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "same\nline count\n")
	hash, err := fileContentHash(path)
	if err != nil {
		t.Fatalf("fileContentHash: %v", err)
	}
	cacheStateFor(t, path, FileState{
		LastReadAt:  time.Now().Add(-2 * time.Second),
		ContentHash: hash,
	})

	if err := checkWriteAllowed(path); err != nil {
		t.Fatalf("expected unchanged content hash to be allowed, got: %v", err)
	}
}

func TestCheckWriteAllowed_ContentHashBlocksSameLineCountExternalEdit(t *testing.T) {
	setupCache(t)
	path := tempFile(t, "alpha\nbeta\n")
	hash, err := fileContentHash(path)
	if err != nil {
		t.Fatalf("fileContentHash: %v", err)
	}
	cacheStateFor(t, path, FileState{
		LastReadAt:  time.Now().Add(1 * time.Hour),
		ContentHash: hash,
	})

	if err := os.WriteFile(path, []byte("gamma\ndelta\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err = checkWriteAllowed(path)
	if err == nil {
		t.Fatal("expected stale hash error after same-line-count external edit")
	}
	if !strings.Contains(err.Error(), "file changed externally") {
		t.Errorf("unexpected error message: %v", err)
	}
}
