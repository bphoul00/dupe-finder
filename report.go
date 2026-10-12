package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)


var reportHTML string

var tailwindCSS string

var reportTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"size": humanSize,
	"date": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("2006-01-02 15:04")
	},
	"unix":  func(t time.Time) int64 { return t.Unix() },
	"lower": strings.ToLower,
	"ext":   fileExt,
	"dirs": func(files []FileEntry) string {
		dirs := make([]string, len(files))
		for i, f := range files {
			dirs[i] = strings.ToLower(f.Dir)
		}
		return strings.Join(dirs, "\n")
	},
	"copies": func(res *ScanResult) int {
		n := 0
		for _, g := range res.Groups {
			n += len(g.Files)
		}
		return n
	},
}).Parse(reportHTML))

// WriteReport renders the scan result as a self-contained HTML file.
func WriteReport(path string, res *ScanResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(f)
	data := struct {
		*ScanResult
		Generated string
		CSS       template.CSS
	}{res, time.Now().Format("2006-01-02 15:04:05"), template.CSS(tailwindCSS)}
	if err := reportTmpl.Execute(bw, data); err != nil {
		f.Close()
		return err
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// fileExt returns the lower-case extension without the dot, or "(none)".
func fileExt(name string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	if ext == "" || "."+ext == strings.ToLower(name) {
		return "(none)" // no extension, or a dotfile like .gitignore
	}
	return ext
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
