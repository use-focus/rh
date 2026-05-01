package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "read":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Usage: rh read <file>")
			os.Exit(1)
		}
		if err := readFile(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "write":
		if len(os.Args) != 5 {
			fmt.Fprintln(os.Stderr, "Usage: rh write <file> <start_hash> <end_hash>")
			os.Exit(1)
		}
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		if err := checkWriteAllowed(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := replaceLines(os.Args[2], os.Args[3], os.Args[4], string(content)); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "grep":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: rh grep <grep_args...>")
			os.Exit(1)
		}
		if err := grepFiles(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "preview":
		if len(os.Args) != 5 {
			fmt.Fprintln(os.Stderr, "Usage: rh preview <file> <start_hash> <end_hash>")
			os.Exit(1)
		}
		if err := previewLines(os.Args[2], os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "append":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Usage: rh append <file>")
			os.Exit(1)
		}
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		if err := checkWriteAllowed(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := appendToFile(os.Args[2], string(content)); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "rh — edit files by stable line hash, not line number.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Each line in a file is assigned a persistent 4-letter hash the first time")
	fmt.Fprintln(os.Stderr, "it is read. Hashes survive edits: lines outside the changed region keep")
	fmt.Fprintln(os.Stderr, "their original hashes across every read, write, and append.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "COMMANDS")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  rh read <file>")
	fmt.Fprintln(os.Stderr, "    Print every line prefixed with its hash.")
	fmt.Fprintln(os.Stderr, "    Always run this first — it registers the file and gives you the")
	fmt.Fprintln(os.Stderr, "    hashes you need for write and preview.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "      abcd line one")
	fmt.Fprintln(os.Stderr, "      efgh line two")
	fmt.Fprintln(os.Stderr, "      ijkl line three")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  rh write <file> <start_hash> <end_hash>")
	fmt.Fprintln(os.Stderr, "    Replace lines from start_hash to end_hash (inclusive) with content")
	fmt.Fprintln(os.Stderr, "    read from stdin. Pass empty stdin to delete the lines.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    On success the output shows 2 lines of context above and below")
	fmt.Fprintln(os.Stderr, "    the edit with their unchanged hashes, plus one of two blocks:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "      <NewLines>")
	fmt.Fprintln(os.Stderr, "        wxyz new content    <- fresh hash, ready to use")
	fmt.Fprintln(os.Stderr, "      </NewLines>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "      <DeletedLines>")
	fmt.Fprintln(os.Stderr, "        efgh line two       <- shown with its old hash for reference")
	fmt.Fprintln(os.Stderr, "      </DeletedLines>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  rh grep <grep_args...>")
	fmt.Fprintln(os.Stderr, "    Run system grep, force filename + line metadata, then replace each")
	fmt.Fprintln(os.Stderr, "    matching line number with that file's stable rh hash.")
	fmt.Fprintln(os.Stderr, "    Matching flags and file args are passed to grep, so -i, -E, -F, -w,")
	fmt.Fprintln(os.Stderr, "    -A, -B, -C, multiple files, and recursive searches work.")
	fmt.Fprintln(os.Stderr, "    Output is grouped into readable blocks:")
	fmt.Fprintln(os.Stderr, "      MATCH - <file>:")
	fmt.Fprintln(os.Stderr, "      <hash> <match or context line>")
	fmt.Fprintln(os.Stderr, "    Every matched file is registered — a write can follow using any")
	fmt.Fprintln(os.Stderr, "    returned hash without needing a separate rh read first.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "      rh grep 'func ' main.go ops.go")
	fmt.Fprintln(os.Stderr, "      rh grep -i -E 'usage|func ' *.go")
	fmt.Fprintln(os.Stderr, "      rh grep -R -A 2 'func main' .")
	fmt.Fprintln(os.Stderr, "      MATCH - main.go:")
	fmt.Fprintln(os.Stderr, "      abcd func main() {")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  rh preview <file> <start_hash> <end_hash>")
	fmt.Fprintln(os.Stderr, "    Show the lines in the given range without modifying the file.")
	fmt.Fprintln(os.Stderr, "    Registers the full file — same as rh read, so a write can follow")
	fmt.Fprintln(os.Stderr, "    directly using any hash in the file, not just the previewed range.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  rh append <file>")
	fmt.Fprintln(os.Stderr, "    Add content from stdin to the end of the file.")
	fmt.Fprintln(os.Stderr, "    Output follows the same format as write (context + <NewLines> block).")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "STDIN")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Content is always supplied via stdin. Use a heredoc to avoid quoting issues:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    rh write file.go abcd efgh << 'EOF'")
	fmt.Fprintln(os.Stderr, "    replacement line")
	fmt.Fprintln(os.Stderr, "    EOF")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    echo 'new line' | rh append file.go")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "GUARDS")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  rh write and rh append are blocked if the file was modified outside")
	fmt.Fprintln(os.Stderr, "  of rh since the last read/grep/preview/write. rh stores a content")
	fmt.Fprintln(os.Stderr, "  checksum, so same-line-count external edits are detected too.")
	fmt.Fprintln(os.Stderr, "  Run rh read, grep, or preview to resync before writing.")
}
