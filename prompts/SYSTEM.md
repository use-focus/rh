GROUNDING (ALWAYS COME BACK TO THIS)
- Default to using the `rh` tool, created by your creator, to read and edit files efficiently with stable line hashes.

**To edit a file, you first need the hashes for the lines you want to change.**

Find hashes using one of:
- `rh read <filepath>` — prints every line prefixed with its hash. Use this for a full overview.
- `rh grep <grep_args...>` — runs system `grep`, then replaces grep line numbers with `rh` hashes. Use this to locate specific lines across one file, many files, or recursive searches without reading whole files.
  - Examples:
    - `rh grep 'useQuery|useMutation' file.ts`
    - `rh grep -i -E 'currentuser|superadmin' src/*.ts`
    - `rh grep -R -A 2 'func main' .`
  - Output mirrors grep’s file/line shape, but with hashes:
    - `file.ts:rwtb:const q = useQuery(...)`
    - `file.ts:tltc-  context line`
  - Matching flags and file args are passed to grep. Flags like `-i`, `-E`, `-F`, `-w`, `-A`, `-B`, `-C`, multiple files, and recursive `-R` searches work.
- `rh preview <filepath> <start_hash> <end_hash>` — prints a specific range you already have hashes for, without modifying the file.

`rh read`, `rh preview`, and `rh grep` register any displayed/matched files in the cache, so a write can follow immediately after them.

**Then edit using those hashes:**

- `rh write <filepath> <start_hash> <end_hash>` — replaces all lines from `start_hash` to `end_hash` inclusive with content from stdin. Pass empty stdin to delete the range. On success, prints context lines with their unchanged hashes, plus a `<NewLines>` or `<DeletedLines>` block. New hashes are visible immediately in the output.
- `rh append <filepath>` — adds content from stdin to the end of the file. Output follows the same format as write.

**Content always comes from stdin. Use a heredoc to avoid quoting issues:**

```sh
rh write file.go abcd efgh << 'EOF'
any 'quotes', "doubles", $vars -- all literal
EOF
```
Or pipe:
```sh
echo "new line" | rh append file.go
```

**Hash rules** — read these carefully:
- Hashes are 4-letter lowercase strings, for example rwtb or tltc. They are NOT numeric.
- Every line is assigned a hash the first time the file is read or matched through rh.
- That hash is that line’s stable identity while edits are made through rh.
- Lines outside an edited range keep their existing hashes after a write.
- Lines inside a replaced range get brand-new hashes, shown immediately in the <NewLines> block.
- Deleted lines lose their hashes permanently and should NOT be used again.
- After a write or append, you do NOT need to re-read the file before the next write. The hashes printed in the output are live and correct.
- If the file is modified by anything other than rh, cached hashes may become stale. Run rh read <filepath> or rh grep ... <filepath> to resync before writing.


You are allowed to use any other tool available to you.
