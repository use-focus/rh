package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// overrideCachePath is set by tests to redirect the cache to a temp location.
var overrideCachePath string

type FileState struct {
	LineHashes  []string  `json:"line_hashes"`  // persistent hash per line, in order
	LastReadAt  time.Time `json:"last_read_at"` // for external modification guard
	ContentHash string    `json:"content_hash,omitempty"`
}

type Cache struct {
	NextID int                  `json:"next_id"` // global counter for allocating new hashes
	Files  map[string]FileState `json:"files"`
}

func cachePath() string {
	if overrideCachePath != "" {
		return overrideCachePath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "rh", "state.json")
}

func loadCache() (*Cache, error) {
	path := cachePath()
	if path == "" {
		return &Cache{Files: make(map[string]FileState)}, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Cache{Files: make(map[string]FileState)}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("corrupt cache: %v", err)
	}
	if c.Files == nil {
		c.Files = make(map[string]FileState)
	}
	return &c, nil
}

func saveCache(c *Cache) error {
	path := cachePath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// allocHash generates the next unique hash using the global counter.
func allocHash(c *Cache) string {
	h := indexToHash(c.NextID)
	c.NextID = (c.NextID + 1) % lcgModulus
	return h
}

// buildHashList generates count new unique hashes advancing the global counter.
func buildHashList(c *Cache, count int) []string {
	hashes := make([]string, count)
	for i := range hashes {
		hashes[i] = allocHash(c)
	}
	return hashes
}

func fileContentHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// checkWriteAllowed returns an error if the file was modified externally since the last read.
func checkWriteAllowed(filePath string) error {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	c, err := loadCache()
	if err != nil {
		// Cache unreadable — don't block.
		return nil
	}
	state, exists := c.Files[abs]
	if !exists {
		// File never read via rh — no state to enforce against.
		return nil
	}

	if state.ContentHash != "" {
		currentHash, err := fileContentHash(abs)
		if err != nil {
			return err
		}
		if currentHash != state.ContentHash {
			return fmt.Errorf(
				"stale hashes: file changed externally since your last rh read/grep/preview/write\nre-read before writing: rh read %s", filePath,
			)
		}
		return nil
	}

	info, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if info.ModTime().After(state.LastReadAt) {
		return fmt.Errorf(
			"stale hashes: file was modified externally since your last read\nrun: rh read %s", filePath,
		)
	}

	return nil
}
