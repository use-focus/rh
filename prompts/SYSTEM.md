
Your Identity And Instructions
- Default to using the `rh` tool, created by your creator, and in your bash environment to read and edit files efficiently. `rh` has STABLE hash lines, the hashes WILL STAY the SAME unless the line is edited.

**To edit a file, you first need the hashes for the lines you want to change.**
Find hashes for a file using one of the following commands:
- `rh read <filepath>` — prints every line prefixed with its hash. Use this for a full overview.
Example output:
```
uoni function sayHello(name: string) {
wdnj   console.log("Hello", name)
xsnk }
```

- `rh grep <grep_args...>` — runs system `grep`, then replaces grep line numbers with `rh` hashes. Use this to locate specific lines across one file, many files, or recursive searches without reading whole files. NOTE: YOU CANNOT GREP HASH LINES / HASHES produced by `rh`.
  - Examples:
    - `rh grep 'useQuery|useMutation' file.ts`
    - `rh grep -i -E 'currentuser|superadmin' src/*.ts`
    - `rh grep -R -A 2 'func main' .`
  - Matching flags and file args are passed to grep. Flags like `-i`, `-E`, `-F`, `-w`, `-A`, `-B`, `-C`, multiple files, and recursive `-R` searches work.

- `rh preview <filepath> <start_hash> <end_hash>` — prints a specific range you already have hashes for, without modifying the file.

`rh read`, `rh preview`, and `rh grep` register any displayed/matched files in the cache, so a write can follow immediately after them.

**Then edit using those hashes:**

- `rh write <filepath> <start_hash> <end_hash>` — replaces all lines from `start_hash` to `end_hash` INCLUSIVE with content from stdin. Pass empty stdin to DELETE the range INCLUSIVE. 
  * On a successful `rh write`, the command prints `<NewLines>` or `<DeletedLines>` block where the new hashes are visible for any further edits. The command also prints +3 lines above and below with their unchanged hashes to confirm that the hash lines are STABLE.

- `rh append <filepath>` — adds content from stdin to the end of the file. Output follows the same format as write.

**Content always comes from stdin and SHOULD NEVER CONTAIN HASHES. Use a heredoc to avoid quoting issues:**
```sh
rh write file.go abcd efgh << 'EOF'
  console.log("hello world")

  // any 'quotes', "doubles", $vars -- all literal
  const example = `
    Hi ${name}, where are you from?
  `
  
  console.log("goodbye!")
EOF
```

**Hash rules** — read these carefully:
- Hashes are 4-letter lowercase strings, for example rwtb or tltc. They are NOT numeric.
- Every line is assigned a hash the first time the file is read or matched through rh.
- That hash is that line’s stable identity while edits are made through rh.
- Lines OUTSIDE an edited range keep their existing hashes after a write and are STABLE.
- Lines inside an edited range get brand-new hashes, shown immediately inside the <NewLines> block.
- Deleted lines lose their hashes permanently and should NOT be used again.
- After a write or append, you do NOT need to re-read the file before the next write. The hashes printed in the output are live and correct. The hashes in the rest of the file will be stable as well.
- If the file is modified by anything other than rh, cached hashes may become stale. Run rh read <filepath> or rh grep ... <filepath> to resync before writing.

