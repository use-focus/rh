
## How to use the `rh` CLI tool:
**Read the hashes for a file using one of the following commands:**
- `rh read <filepath>` — prints every line prefixed with its hash. Use this for a full overview.
Example output:
```
uoni function sayHello(name: string) {
wdnj   console.log("Hello", name)
xsnk }
```

**To search for a string and find the hashes near it use:**
- `rh rg <rg_args...>` runs `rg` and replaces line numbers with `rh` hashes. Use it to search one or more files. NOTE: YOU CANNOT SEARCH HASH LINES PRODUCED BY `rh`.
  - Examples:
    - `rh rg 'useQuery|useMutation' file.ts`
    - `rh rg -i 'currentuser|superadmin' src/*.ts`
    - `rh rg -A 2 'func main' .`
  - `rh` passes the arguments to `rg`. Flags such as `-i`, `-F`, `-w`, `-A`, `-B`, and `-C` work.
  - The search fails if the `rg` output exceeds 16,000 characters. This value estimates 4,000 tokens.

**To read / preview between 2 hashes use:**
- `rh preview <filepath> <start_hash> <end_hash>` — prints a specific range you already have hashes for, without modifying the file.

`rh read`, `rh preview`, and `rh rg` register displayed or matched files in the cache. You can then write to these files.

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
- If another tool modifies the file, the cached hashes can become stale. Run rh read <filepath> or rh rg ... <filepath> again.


