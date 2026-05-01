package main

import "fmt"

const diffContextLines = 3

// printWriteResult prints the result of a write or append to stdout.
// Context lines (unchanged) are shown with their existing hashes.
//
// Two block types are used depending on what happened in the replaced range:
//   - <NewLines>     — new or replacement content, shown with freshly allocated hashes.
//   - <DeletedLines> — pure deletion (empty new content), shown with the old hashes that were removed.
func printWriteResult(oldLines, oldHashes []string, startIdx, endIdx int, newLines, newHashes []string) {
	ctxStart := startIdx - diffContextLines
	if ctxStart < 0 {
		ctxStart = 0
	}

	// Context above — unchanged lines with their existing hashes.
	for i := ctxStart; i < startIdx && i < len(oldLines); i++ {
		hash := ""
		if i < len(oldHashes) {
			hash = oldHashes[i] + " "
		}
		fmt.Printf("  %s%s\n", hash, oldLines[i])
	}

	if len(newLines) == 0 {
		// Pure deletion — show the lines that were removed with their old hashes.
		fmt.Println("<DeletedLines>")
		endCap := endIdx + 1
		if endCap > len(oldLines) {
			endCap = len(oldLines)
		}
		for _, line := range oldLines[startIdx:endCap] {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println("</DeletedLines>")
	} else {
		// New or replacement content — show incoming lines with their fresh hashes.
		fmt.Println("<NewLines>")
		for i, line := range newLines {
			fmt.Printf("  %s %s\n", newHashes[i], line)
		}
		fmt.Println("</NewLines>")
	}

	// Context below — lines after the replaced region, with their existing hashes.
	afterStart := endIdx + 1
	afterEnd := afterStart + diffContextLines
	if afterEnd > len(oldLines) {
		afterEnd = len(oldLines)
	}
	for i := afterStart; i < afterEnd; i++ {
		hash := ""
		if i < len(oldHashes) {
			hash = oldHashes[i] + " "
		}
		fmt.Printf("  %s%s\n", hash, oldLines[i])
	}
}
