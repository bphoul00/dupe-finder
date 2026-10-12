// dupe-finder scans a folder for files that share the same name and writes
// an HTML report listing every location of each duplicated name.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	out := flag.String("o", "report.html", "output HTML file")
	ignoreCase := flag.Bool("ignore-case", false, "treat names that differ only in case as the same")
	exclude := flag.String("exclude", "", "comma-separated directory names to skip (e.g. .git,node_modules)")
	workers := flag.Int("workers", 0, "number of parallel workers (default: 2 x CPU cores)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [folder]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	res, err := Scan(root, Options{
		IgnoreCase: *ignoreCase,
		Exclude:    parseExclude(*exclude),
		Workers:    *workers,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := WriteReport(*out, res); err != nil {
		fmt.Fprintln(os.Stderr, "error writing report:", err)
		os.Exit(1)
	}

	copies := 0
	for _, g := range res.Groups {
		copies += len(g.Files)
	}
	fmt.Printf("Scanned %d files in %d folders under %s in %s\n",
		res.FilesSeen, res.DirsSeen, res.Root, res.Duration.Round(1e6))
	fmt.Printf("Found %d duplicated names (%d files)\n", len(res.Groups), copies)
	if len(res.Warnings) > 0 {
		fmt.Printf("%d folders could not be read (listed in the report)\n", len(res.Warnings))
	}
	fmt.Println("Report written to", *out)
}

func parseExclude(s string) map[string]bool {
	m := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			m[part] = true
		}
	}
	return m
}
