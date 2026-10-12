package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Options controls how a folder is scanned.
type Options struct {
	IgnoreCase bool
	Exclude    map[string]bool // directory names to skip
	Workers    int
}

// FileEntry is one copy of a duplicated file.
type FileEntry struct {
	Path    string
	Dir     string
	Size    int64
	ModTime time.Time
}

// Group is a filename that appears in more than one place.
type Group struct {
	Name  string
	Files []FileEntry
}

// ScanResult holds everything found by a scan.
type ScanResult struct {
	Root       string
	FilesSeen  int
	DirsSeen   int
	Groups     []Group
	Warnings   []string
	Duration   time.Duration
	IgnoreCase bool
}

// walker hands out directories to a fixed pool of workers. pending counts
// directories that are queued or still being read; when it hits zero the walk is done.
type walker struct {
	opts    Options
	mu      sync.Mutex
	cond    *sync.Cond
	queue   []string
	pending int

	warnMu   sync.Mutex
	warnings []string
}

type workerResult struct {
	byName map[string][]string
	names  map[string]string // key -> display name (first spelling seen)
	files  int
	dirs   int
}

// Scan walks root in parallel and groups files that share a name.
func Scan(root string, opts Options) (*ScanResult, error) {
	start := time.Now()
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "scan", Path: abs, Err: os.ErrInvalid}
	}
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU() * 2
	}

	w := &walker{opts: opts, queue: []string{abs}, pending: 1}
	w.cond = sync.NewCond(&w.mu)

	results := make([]workerResult, opts.Workers)
	var wg sync.WaitGroup
	for i := range opts.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = w.work()
		}()
	}
	wg.Wait()

	// Merge the per-worker maps.
	byName := map[string][]string{}
	names := map[string]string{}
	res := &ScanResult{Root: abs, IgnoreCase: opts.IgnoreCase}
	for _, r := range results {
		res.FilesSeen += r.files
		res.DirsSeen += r.dirs
		for k, paths := range r.byName {
			byName[k] = append(byName[k], paths...)
			if _, ok := names[k]; !ok {
				names[k] = r.names[k]
			}
		}
	}

	for k, paths := range byName {
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		g := Group{Name: names[k]}
		for _, p := range paths {
			fe := FileEntry{Path: p, Dir: filepath.Dir(p)}
			if fi, err := os.Lstat(p); err == nil {
				fe.Size = fi.Size()
				fe.ModTime = fi.ModTime()
			}
			g.Files = append(g.Files, fe)
		}
		res.Groups = append(res.Groups, g)
	}
	sort.Slice(res.Groups, func(i, j int) bool {
		a, b := res.Groups[i], res.Groups[j]
		if len(a.Files) != len(b.Files) {
			return len(a.Files) > len(b.Files)
		}
		return a.Name < b.Name
	})

	sort.Strings(w.warnings)
	res.Warnings = w.warnings
	res.Duration = time.Since(start)
	return res, nil
}

func (w *walker) work() workerResult {
	r := workerResult{byName: map[string][]string{}, names: map[string]string{}}
	for {
		w.mu.Lock()
		for len(w.queue) == 0 && w.pending > 0 {
			w.cond.Wait()
		}
		if w.pending == 0 {
			w.mu.Unlock()
			return r
		}
		dir := w.queue[len(w.queue)-1]
		w.queue = w.queue[:len(w.queue)-1]
		w.mu.Unlock()

		subdirs := w.readDir(dir, &r)

		w.mu.Lock()
		w.queue = append(w.queue, subdirs...)
		w.pending += len(subdirs) - 1
		wake := w.pending == 0 || len(subdirs) > 0
		w.mu.Unlock()
		if wake {
			w.cond.Broadcast()
		}
	}
}

func (w *walker) readDir(dir string, r *workerResult) []string {
	r.dirs++
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.warn(err.Error())
		// ReadDir may still return the entries it managed to read.
	}
	var subdirs []string
	for _, e := range entries {
		name := e.Name()
		typ := e.Type()
		switch {
		case typ&os.ModeSymlink != 0:
			continue // never follow symlinks: avoids loops and double counting
		case e.IsDir():
			if !w.opts.Exclude[name] {
				subdirs = append(subdirs, filepath.Join(dir, name))
			}
		default:
			r.files++
			key := name
			if w.opts.IgnoreCase {
				key = strings.ToLower(name)
			}
			if _, ok := r.names[key]; !ok {
				r.names[key] = name
			}
			r.byName[key] = append(r.byName[key], filepath.Join(dir, name))
		}
	}
	return subdirs
}

func (w *walker) warn(msg string) {
	w.warnMu.Lock()
	w.warnings = append(w.warnings, msg)
	w.warnMu.Unlock()
}
