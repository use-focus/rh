package main

import (
	"strings"
	"testing"
)

// ── indexToHash ───────────────────────────────────────────────────────────────

func TestIndexToHash_FourLowercaseChars(t *testing.T) {
	for _, idx := range []int{0, 1, 100, 99999} {
		h := indexToHash(idx)
		if len(h) != 4 {
			t.Errorf("indexToHash(%d) = %q: want 4 chars, got %d", idx, h, len(h))
		}
		for _, c := range h {
			if c < 'a' || c > 'z' {
				t.Errorf("indexToHash(%d) = %q: contains non-lowercase char %c", idx, h, c)
			}
		}
	}
}

func TestIndexToHash_Deterministic(t *testing.T) {
	for _, idx := range []int{0, 7, 42, 1000} {
		first := indexToHash(idx)
		second := indexToHash(idx)
		if first != second {
			t.Errorf("indexToHash(%d) is not deterministic: %q != %q", idx, first, second)
		}
	}
}

func TestIndexToHash_UniqueAcrossRange(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		h := indexToHash(i)
		if seen[h] {
			t.Errorf("duplicate hash %q at index %d", h, i)
		}
		seen[h] = true
	}
}

// ── findHashIndex ─────────────────────────────────────────────────────────────

func TestFindHashIndex_Found(t *testing.T) {
	hashes := []string{"aaaa", "bbbb", "cccc", "dddd"}
	idx, err := findHashIndex(hashes, "cccc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 2 {
		t.Errorf("expected index 2, got %d", idx)
	}
}

func TestFindHashIndex_FirstElement(t *testing.T) {
	hashes := []string{"aaaa", "bbbb", "cccc"}
	idx, err := findHashIndex(hashes, "aaaa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestFindHashIndex_LastElement(t *testing.T) {
	hashes := []string{"aaaa", "bbbb", "cccc"}
	idx, err := findHashIndex(hashes, "cccc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 2 {
		t.Errorf("expected index 2, got %d", idx)
	}
}

func TestFindHashIndex_NotFound(t *testing.T) {
	hashes := []string{"aaaa", "bbbb", "cccc"}
	_, err := findHashIndex(hashes, "zzzz")
	if err == nil {
		t.Fatal("expected error for missing hash, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestFindHashIndex_EmptyList(t *testing.T) {
	_, err := findHashIndex([]string{}, "aaaa")
	if err == nil {
		t.Fatal("expected error for empty hash list, got nil")
	}
}

func TestFindHashIndex_InvalidTooShort(t *testing.T) {
	_, err := findHashIndex([]string{"aaaa"}, "abc")
	if err == nil {
		t.Fatal("expected error for short hash, got nil")
	}
	if !strings.Contains(err.Error(), "4 lowercase") {
		t.Errorf("expected format error, got: %v", err)
	}
}

func TestFindHashIndex_InvalidTooLong(t *testing.T) {
	_, err := findHashIndex([]string{"aaaa"}, "aaaaa")
	if err == nil {
		t.Fatal("expected error for long hash, got nil")
	}
}

func TestFindHashIndex_InvalidNonAlpha(t *testing.T) {
	_, err := findHashIndex([]string{"aaaa"}, "ab1c")
	if err == nil {
		t.Fatal("expected error for non-alpha hash, got nil")
	}
	if !strings.Contains(err.Error(), "4 lowercase") {
		t.Errorf("expected format error, got: %v", err)
	}
}

func TestFindHashIndex_InvalidUppercase(t *testing.T) {
	_, err := findHashIndex([]string{"aaaa"}, "AAAA")
	if err == nil {
		t.Fatal("expected error for uppercase hash, got nil")
	}
}
