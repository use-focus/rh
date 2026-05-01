package main

import "fmt"

const (
	lcgModulus    = 456976 // 26^4
	lcgMultiplier = 27717
	lcgIncrement  = 314159
)

// indexToHash converts an integer to a pseudo-random 4-letter string (a-z).
// Used by allocHash to generate new unique hashes from the global counter.
func indexToHash(index int) string {
	mapped := int((int64(index)*lcgMultiplier + lcgIncrement) % lcgModulus)
	if mapped < 0 {
		mapped += lcgModulus
	}
	chars := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		chars[i] = byte('a' + (mapped % 26))
		mapped /= 26
	}
	return string(chars)
}

// findHashIndex returns the position of hash in the file's LineHashes list.
// Returns a clear error if the hash format is invalid or the hash isn't found.
func findHashIndex(hashes []string, hash string) (int, error) {
	if len(hash) != 4 {
		return 0, fmt.Errorf("invalid hash %q: must be exactly 4 lowercase letters", hash)
	}
	for _, c := range hash {
		if c < 'a' || c > 'z' {
			return 0, fmt.Errorf("invalid hash %q: must be exactly 4 lowercase letters", hash)
		}
	}
	for i, h := range hashes {
		if h == hash {
			return i, nil
		}
	}
	return 0, fmt.Errorf("hash %q not found — run: rh read <file>", hash)
}

// isFormatError reports whether hash fails the 4-lowercase-letter format check.
// Used to distinguish format errors from "not found" errors before opening files.
func isFormatError(hash string) bool {
	if len(hash) != 4 {
		return true
	}
	for _, c := range hash {
		if c < 'a' || c > 'z' {
			return true
		}
	}
	return false
}
