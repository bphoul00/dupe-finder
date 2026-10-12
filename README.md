# dupe-finder

Scans a folder (and all its subfolders) for files that share the same name and writes an
HTML report listing every location of each duplicated name.

It compares only file names, never file contents, and reads folders in parallel.

## Build

```sh
go build -o dupe-finder .
```

## Usage

```sh
./dupe-finder [flags] [folder]      # folder defaults to "."
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-o` | `report.html` | Output HTML file |
| `-ignore-case` | off | Treat `Report.PDF` and `report.pdf` as the same name |
| `-exclude` | none | Comma-separated folder names to skip, e.g. `.git,node_modules` |
| `-workers` | 2 × CPU cores | Number of folders read in parallel |

Example:

```sh
./dupe-finder -exclude .git,node_modules -o dupes.html ~/directory/duplicate_directory
```

Notes:
- Names must match exactly, including the extension (`report.pdf` and `report.txt` are different).
- Symlinks are skipped, so links can't cause loops or double counting.
- Folders that can't be read are listed at the bottom of the report.

## Using the report

Open the HTML file in a browser.

**Filters** (panel at the top): narrow the list by file name, folder, minimum number of copies,
and file type. File type chips are in alphabetical order. **All** (on by default) shows every type. Clicking a type such as `.go` turns All off and shows only the types you pick. With nothing selected, nothing shows.
"Only show groups with selected files" lets you review what you've ticked.

**Same size only**: tick this to count files as duplicates only when the name *and* the exact size match.
Copies with a different size are hidden, and groups with no size match disappear.
Copies whose size matches another copy are always tagged "same size", even with this off.

**Choosing what to delete:**
- Tick the checkbox next to any copy.
- Tick the checkbox in a group's table header to select every copy of that name
  (with "Same size only" on, only the same-size copies).
- Or use a group's buttons to tick every copy except one: **Keep newest**, **Keep oldest**,
  **Keep largest**, **Keep first**, or **Keep only this** on a specific row. **Clear** unticks the group.
- If you tick every copy of a name (or every same-size copy while "Same size only" is on),
  that group is outlined in red and the bottom bar warns you.

**Deleting:** a bar at the bottom shows how many files are selected and their total size.
- **Copy command** copies a `\rm -fv -- '...'` command to paste into your terminal. It deletes everything without asking: the `\` skips any `rm -i` alias, and `-f` skips the write-protected prompts.
- **Download .sh** saves it as a script.
- **Show command** lets you review it first.
- Tick **Move to trash instead** to use `gio trash` so files can be restored.

The page itself never deletes anything. Paths are shell-quoted, so spaces and apostrophes are safe.
If the folder changed after the scan, re-run `dupe-finder` before deleting.

## Changing the page design

The page layout lives in `templates/report.html.tmpl` and is styled with [Tailwind CSS](https://tailwindcss.com).
The compiled CSS (`templates/tailwind.css`) is built into the program, so the report works offline and
a plain `go build` doesn't need Node. If you change classes in the template or `web/input.css`, rebuild the CSS first:

```sh
cd web && npm install && cd ..   # once
go generate && go build -o dupe-finder .
```

## Test

```sh
go test -race ./...
```
