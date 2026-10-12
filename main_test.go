package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeTree creates the given relative file paths under a temp directory.
func makeTree(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func groupsByName(res *ScanResult) map[string]int {
	m := map[string]int{}
	for _, g := range res.Groups {
		m[g.Name] = len(g.Files)
	}
	return m
}

func TestScanExactNames(t *testing.T) {
	root := makeTree(t,
		"a/report.pdf", "b/report.pdf", "c/d/report.pdf",
		"a/report.txt",             // different extension: not a match
		"a/Notes.md", "b/notes.md", // different case: not a match by default
		"unique.go",
	)
	res, err := Scan(root, Options{Workers: 4})
	if err != nil {
		t.Fatal(err)
	}
	got := groupsByName(res)
	if len(got) != 1 || got["report.pdf"] != 3 {
		t.Fatalf("got groups %v, want only report.pdf x3", got)
	}
	if res.FilesSeen != 7 {
		t.Errorf("FilesSeen = %d, want 7", res.FilesSeen)
	}
}

func TestScanIgnoreCase(t *testing.T) {
	root := makeTree(t, "a/Notes.md", "b/notes.md", "c/NOTES.MD", "d/other.md")
	res, err := Scan(root, Options{IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 1 || len(res.Groups[0].Files) != 3 {
		t.Fatalf("got groups %v, want one group of 3", groupsByName(res))
	}
}

func TestScanExclude(t *testing.T) {
	root := makeTree(t, "src/index.js", "node_modules/pkg/index.js", "lib/index.js", "node_modules/x.js")
	res, err := Scan(root, Options{Exclude: map[string]bool{"node_modules": true}})
	if err != nil {
		t.Fatal(err)
	}
	got := groupsByName(res)
	if got["index.js"] != 2 {
		t.Fatalf("got %v, want index.js x2", got)
	}
	for _, f := range res.Groups[0].Files {
		if strings.Contains(f.Path, "node_modules") {
			t.Errorf("excluded path reported: %s", f.Path)
		}
	}
}

func TestScanSkipsSymlinks(t *testing.T) {
	root := makeTree(t, "a/file.txt", "b/file.txt")
	if err := os.Symlink(filepath.Join(root, "a"), filepath.Join(root, "loop")); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	res, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := groupsByName(res)["file.txt"]; got != 2 {
		t.Fatalf("file.txt copies = %d, want 2 (symlinked dir must not be followed)", got)
	}
}

func TestScanSortsByCopies(t *testing.T) {
	root := makeTree(t, "a/x", "b/x", "a/y", "b/y", "c/y")
	res, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 2 || res.Groups[0].Name != "y" || res.Groups[1].Name != "x" {
		t.Fatalf("unexpected order: %+v", res.Groups)
	}
}

func TestWriteReport(t *testing.T) {
	root := makeTree(t, "a/<b>.txt", "c/<b>.txt")
	res, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "report.html")
	if err := WriteReport(out, res); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	if !strings.Contains(html, "&lt;b&gt;.txt") || strings.Contains(html, "<b>.txt") {
		t.Error("file names are not HTML-escaped")
	}
	if !strings.Contains(html, "2 copies") {
		t.Error("report missing copy count")
	}
}

func TestWriteReportSelectionData(t *testing.T) {
	root := makeTree(t, "a b/it's a file.txt", "c/it's a file.txt")
	res, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "report.html")
	if err := WriteReport(out, res); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	want := `data-path="` + filepath.Join(root, "a b", "it&#39;s a file.txt") + `"`
	if !strings.Contains(html, want) {
		t.Errorf("missing escaped path attribute %s", want)
	}
	if n := strings.Count(html, `aria-label="Select for deletion"`); n != 2 {
		t.Errorf("got %d row checkboxes, want 2", n)
	}
	if !strings.Contains(html, `data-ext="txt"`) {
		t.Error("missing data-ext on group")
	}
	if !strings.Contains(html, `aria-label="Select all copies"`) {
		t.Error("missing per-group select-all checkbox")
	}
	if !strings.Contains(html, `id="fsize"`) {
		t.Error("missing same-size checkbox")
	}
	if !strings.Contains(html, "--tw-") || !strings.Contains(html, "tr.selected") {
		t.Error("Tailwind CSS is not inlined")
	}
}

func TestFileExt(t *testing.T) {
	for name, want := range map[string]string{
		"a.PDF": "pdf", "Makefile": "(none)", ".gitignore": "(none)", "x.tar.gz": "gz", ".env.local": "local",
	} {
		if got := fileExt(name); got != want {
			t.Errorf("fileExt(%q) = %q, want %q", name, got, want)
		}
	}
}
