# rh — agent file editing CLI tool

`rh` is designed for efficient input and output from AI agents by minimizing reads and reducing output tokens for writing. 

For best usage, add the [system prompt](prompts/SYSTEM.md) to your agent.

## Install
Requires Go 1.25+.

**Install directly:**

> Requires `~/go/bin` (or `$GOPATH/bin`) to be on your `$PATH`. If `rh` isn't found after install, add `export PATH="$PATH:$(go env GOPATH)/bin"` to your shell profile.

```sh
go install github.com/use-focus/rh@latest
```

**Or clone and build:**

```sh
git clone https://github.com/use-focus/rh
cd rh
make install
```

## Commands

| Command | Description |
|---|---|
| `rh read <file>` | Print every line prefixed with its hash. Run this first to register a file. |
| `rh grep <grep_args...>` | Run grep and replace line numbers with stable hashes. Registered files are ready to write immediately. |
| `rh preview <file> <start_hash> <end_hash>` | Show a line range without modifying the file. Registers the file for subsequent writes. |
| `rh write <file> <start_hash> <end_hash>` | Replace lines from `start_hash` to `end_hash` (inclusive) with content from stdin. Empty stdin deletes the range. |
| `rh append <file>` | Add content from stdin to the end of the file. |

## Usage

**Read a file to get hashes:**

```sh
rh read main.go
# abcd func main() {
# efgh     fmt.Println("hello")
# ijkl }
```

**Write (replace) a range using a heredoc:**

```sh
rh write main.go efgh efgh << 'EOF'
    fmt.Println("world")
EOF
```

**Delete a range (empty stdin):**

```sh
echo -n | rh write main.go efgh ijkl
```

**Append content:**

```sh
echo "// end of file" | rh append main.go
```

**Grep across files:**

```sh
rh grep -R 'func ' .
rh grep -i -E 'usage|error' *.go
```

## Guards

`rh write` and `rh append` are blocked if the file was modified outside of `rh` since the last read/grep/preview/write. Run `rh read`, `rh grep`, or `rh preview` to resync before writing.
