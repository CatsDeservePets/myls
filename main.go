package main

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// tabWidth is the tab stop width in spaces.
const tabWidth = 8

// An entry is a file or directory being listed.
type entry struct {
	name      string      // display name (may be relative)
	path      string      // absolute path
	info      os.FileInfo // file metadata
	gitStatus string      // Git status (long mode only)
}

// sortBy controls the primary sort key.
type sortBy int

const (
	name sortBy = iota
	extension
	size
	mtime
	git
)

// Set implements the [flag.Value] interface.
func (s *sortBy) Set(val string) error {
	switch val {
	case "name":
		*s = name
	case "ext", "extension":
		*s = extension
	case "size":
		*s = size
	case "time", "mtime":
		*s = mtime
	case "git":
		*s = git
	default:
		return errors.New("must be name, extension, size, time, or git")
	}
	return nil
}

// String implements the [flag.Value] interface.
func (s sortBy) String() string {
	switch s {
	case name:
		return "name"
	case extension:
		return "extension"
	case size:
		return "size"
	case mtime:
		return "time"
	case git:
		return "git"
	default:
		return ""
	}
}

var (
	progName   = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	homeDir, _ = os.UserHomeDir()
	currYear   = time.Now().Year()

	gitRepos   = map[string]map[string]string{}
	gitReposMu sync.Mutex
)

func main() {
	initOptions()

	files, dirs := collectEntries(flag.Args())
	if len(dirs) == 0 && len(files) == 0 {
		os.Exit(1)
	}
	showDirHeader := len(files) > 0 || len(dirs) > 1

	if opt.long && opt.git {
		attachGitToFiles(files)
	}
	sortEntries(files)
	printEntries(files)

	sortEntries(dirs)

	dirEntries := make([][]entry, len(dirs))
	var wg sync.WaitGroup

	for i, d := range dirs {
		wg.Go(func() {
			ents, err := readDir(d.path)
			if err != nil {
				showError(err)
				dirEntries[i] = nil
				return
			}
			if opt.all {
				ents = append(selfAndParent(d.path), ents...)
			} else {
				ents = slices.DeleteFunc(ents, isHidden)
			}
			if opt.long && opt.git {
				attachGitToDir(d.path, ents)
			}
			sortEntries(ents)
			dirEntries[i] = ents
		})
	}

	wg.Wait()

	for i, d := range dirs {
		if i > 0 || len(files) > 0 {
			// Separate directory listing from previous output.
			fmt.Println()
		}
		if showDirHeader {
			// If output has multiple sections, label directory
			// using the user-supplied path (abbreviated with ~).
			fmt.Printf("%s:\n", tildePath(d.name))
		}
		printEntries(dirEntries[i])
	}
}

// collectEntries expands args and splits them into files and directories.
func collectEntries(args []string) (files, dirs []entry) {
	if len(args) == 0 {
		args = []string{"."}
	}

	for _, pattern := range args {
		// Windows does not expand shell globs automatically,
		// so we start by treating patterns as literal paths.
		paths := []string{pattern}
		// Override the literal path when globbing succeeds.
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			paths = matches
		}

		for _, p := range paths {
			info, err := os.Lstat(p)
			if err != nil {
				showError(err)
				continue
			}

			abs := p
			if a, err := filepath.Abs(p); err == nil {
				abs = a
			}
			ent := entry{
				name: p,
				path: abs,
				info: info,
			}
			if !opt.dir && info.IsDir() {
				// Prefer entry type over string to simplify sorting.
				dirs = append(dirs, ent)
			} else {
				files = append(files, ent)
			}
		}
	}
	return files, dirs
}

// sortEntries sorts ents according to the active sort and grouping options.
func sortEntries(ents []entry) {
	// Always sort by name first.
	slices.SortFunc(ents, func(a, b entry) int {
		if opt.reverse {
			return strings.Compare(strings.ToLower(b.name), strings.ToLower(a.name))
		}
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})

	switch opt.sort {
	case extension:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if opt.reverse {
				return strings.Compare(strings.ToLower(filepath.Ext(b.name)), strings.ToLower(filepath.Ext(a.name)))
			}
			return strings.Compare(strings.ToLower(filepath.Ext(a.name)), strings.ToLower(filepath.Ext(b.name)))
		})
	case size:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if opt.reverse {
				return cmp.Compare(b.info.Size(), a.info.Size())
			}
			return cmp.Compare(a.info.Size(), b.info.Size())
		})
	case mtime:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if opt.reverse {
				return b.info.ModTime().Compare(a.info.ModTime())
			}
			return a.info.ModTime().Compare(b.info.ModTime())
		})
	case git:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if opt.reverse {
				return strings.Compare(strings.ToLower(b.gitStatus), strings.ToLower(a.gitStatus))
			}
			return strings.Compare(strings.ToLower(a.gitStatus), strings.ToLower(b.gitStatus))
		})
	}

	if opt.dirsFirst {
		slices.SortStableFunc(ents, func(a, b entry) int {
			ad, bd := isDir(a), isDir(b)
			switch {
			case ad == bd:
				return 0
			case ad:
				return -1
			default:
				return 1
			}
		})
	}
}

// selfAndParent returns entries for "." and ".." within dir.
func selfAndParent(dir string) []entry {
	ents := make([]entry, 0, 2)
	for _, name := range [...]string{".", ".."} {
		full := filepath.Join(dir, name)
		if info, err := os.Lstat(full); err != nil {
			showError(err)
		} else {
			ents = append(ents, entry{
				name: name,
				path: full,
				info: info,
			})
		}
	}
	return ents
}

// readDir is like [os.ReadDir], but returns a slice of [entry] rather than
// [os.DirEntry] and does not sort by filename.
func readDir(path string) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dirents, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	ents := make([]entry, 0, len(dirents))
	for _, de := range dirents {
		info, err := de.Info()
		if err != nil {
			showError(err)
			continue
		}
		name := de.Name()
		full := filepath.Join(path, name)
		ents = append(ents, entry{
			name: name,
			path: full,
			info: info,
		})
	}

	return ents, nil
}

// readDirNames is a convenience wrapper for [os.File.Readdirnames].
func readDirNames(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Readdirnames(-1)
}

// attachGitToFiles populates gitStatus for ents, doing at most one lookup
// per directory.
func attachGitToFiles(ents []entry) {
	dirCache := make(map[string]map[string]string)
	showGit := false

	for i := range ents {
		e := &ents[i]
		var dir string
		if e.info.IsDir() {
			// For directory entries, use directory itself; it may be the repo root.
			dir = e.path
		} else {
			dir = filepath.Dir(e.path)
		}

		stats, ok := dirCache[dir]
		if !ok {
			stats = gitStatusesForDir(dir)
			dirCache[dir] = stats
		}
		if stats == nil {
			continue
		}
		showGit = true
		if signs, ok := stats[e.path]; ok {
			e.gitStatus = strings.ReplaceAll(signs, " ", "-")
		}
	}

	// Only add placeholders if any Git status is present.
	if !showGit {
		return
	}
	for i := range ents {
		if ents[i].gitStatus == "" {
			ents[i].gitStatus = "--"
		}
	}
}

// attachGitToDir populates gitStatus for ents using dir's repository.
func attachGitToDir(dir string, ents []entry) {
	stats := gitStatusesForDir(dir)
	if stats == nil {
		return
	}

	for i := range ents {
		e := &ents[i]
		if signs, ok := stats[e.path]; ok {
			e.gitStatus = strings.ReplaceAll(signs, " ", "-")
		} else {
			e.gitStatus = "--"
		}
	}
}

// gitStatusesForDir returns Git status codes for dir's repository.
// The map keys are absolute paths for all entries reported by Git.
// It returns nil if no repository is found or status cannot be read.
func gitStatusesForDir(dir string) map[string]string {
	priority := func(signs string) int {
		switch signs {
		case "!!":
			return 1
		case "??":
			return 2
		default:
			return 3
		}
	}

	root := gitRoot(dir)
	if root == "" {
		return nil
	}

	gitReposMu.Lock()
	if st, ok := gitRepos[root]; ok {
		gitReposMu.Unlock()
		return st
	}
	gitReposMu.Unlock()

	cmd := exec.Command(
		"git", "-C", root,
		"status", "--porcelain=v1", "-z", "--ignored",
	)
	out, err := cmd.Output()
	if err != nil {
		gitReposMu.Lock()
		gitRepos[root] = nil
		gitReposMu.Unlock()
		return nil
	}

	stats := make(map[string]string)
	for rec := range bytes.SplitSeq(out, []byte{0}) {
		// skip invalid status (e.g. second part of rename entry)
		if len(rec) < 4 || rec[2] != ' ' {
			continue
		}
		signs := string(rec[:2])
		rel := string(rec[3:])
		rel = filepath.FromSlash(rel)
		full := filepath.Join(root, rel)

		if prev, ok := stats[full]; !ok || priority(prev) < priority(signs) {
			stats[full] = signs
		}

		// propagate "highest" status to all parent dirs
		dirPath := filepath.Dir(full)
		for len(dirPath) >= len(root) {
			prev, ok := stats[dirPath]
			if !ok || priority(prev) < priority(signs) {
				stats[dirPath] = signs
			}
			if dirPath == root {
				break
			}
			parent := filepath.Dir(dirPath)
			if parent == dirPath {
				break
			}
			dirPath = parent
		}
	}

	gitReposMu.Lock()
	gitRepos[root] = stats
	gitReposMu.Unlock()

	return stats
}

// gitRoot returns the repository root containing dir, or "" if none is found.
func gitRoot(dir string) string {
	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			return ""
		}
		root = parent
	}
}

// printEntries prints ents using the active output mode.
func printEntries(ents []entry) {
	if len(ents) == 0 {
		return
	}
	switch {
	case opt.long:
		printLong(ents)
	case opt.oneEntry:
		print1PerLine(ents)
	default:
		printShort(ents)
	}
}

// A row holds the formatted columns of an [entry] in long output.
type row struct {
	modeStr string
	sizeStr string
	timeStr string
	gitStr  string
	nameStr string
}

// printLong prints ents with metadata columns and aligns them by content width.
func printLong(ents []entry) {
	rows := make([]row, 0, len(ents))

	sizeWidth := 0
	timeWidth := 0
	gitWidth := len(ents[0].gitStatus) // Always 0 or 2 for every entry.

	// Format once; print aligned after widths are known.
	for _, e := range ents {
		name := e.name
		if suffix := classify(e); suffix != 0 {
			name += string(suffix)
			if suffix == '@' {
				if target, err := os.Readlink(e.path); err != nil {
					showError(err)
				} else {
					name += " -> " + target
				}
			}
		}

		var sizeStr string
		if isDir(e) {
			if children, err := readDirNames(e.path); err != nil {
				sizeStr = "!"
			} else {
				sizeStr = fmt.Sprintf("%d", len(children))
			}
		} else {
			sizeStr = humanReadable(e.info.Size())
		}
		if n := len(sizeStr); n > sizeWidth {
			sizeWidth = n
		}

		timeStr := formatTime(e.info.ModTime())
		if n := len(timeStr); n > timeWidth {
			timeWidth = n
		}

		rows = append(rows, row{
			modeStr: mode(e),
			sizeStr: sizeStr,
			timeStr: timeStr,
			gitStr:  e.gitStatus,
			nameStr: name,
		})
	}

	if gitWidth > 0 {
		gitWidth++ // needs separation if visible
	}
	for _, r := range rows {
		fmt.Printf("%s %*s %-*s%*s %s\n",
			r.modeStr,
			sizeWidth, r.sizeStr,
			timeWidth, r.timeStr,
			gitWidth, r.gitStr,
			r.nameStr,
		)
	}
}

// print1PerLine prints each entry in ents on its own line.
func print1PerLine(ents []entry) {
	for _, e := range ents {
		name := e.name
		if suffix := classify(e); suffix != 0 {
			name += string(suffix)
		}
		fmt.Println(name)
	}
}

// printShort prints ents in tab-aligned columns, filling top-to-bottom first
// and then left-to-right.
func printShort(ents []entry) {
	entryCount := len(ents)
	names := make([]string, entryCount)
	nameWidth := 0

	for i, e := range ents {
		name := e.name
		if n := len(name); n > nameWidth {
			nameWidth = n
		}
		if suffix := classify(e); suffix != 0 {
			name += string(suffix)
		}
		names[i] = name
	}

	nameWidth += 1 // Account for (possible) classification
	colTabs := nameWidth/tabWidth + 1
	cols := min(max(opt.termWidth/(colTabs*tabWidth), 1), entryCount)

	if cols == 1 {
		for _, n := range names {
			fmt.Println(n)
		}
		return
	}

	rows := (entryCount + cols - 1) / cols

	for r := range rows {
		for c := range cols {
			i := c*rows + r
			if i >= entryCount {
				break
			}

			s := names[i]
			fmt.Print(s)

			if c == cols-1 || i+rows >= entryCount {
				continue
			}

			tabs := max(colTabs-len(s)/tabWidth, 1)
			fmt.Print(strings.Repeat("\t", tabs))
		}
		fmt.Println()
	}
}

// isDir is like [os.FileInfo.IsDir], but also follows symlinks.
// It is currently only used for better dircounts and directory grouping.
func isDir(e entry) bool {
	if e.info.IsDir() {
		return true
	}
	if e.info.Mode()&os.ModeSymlink != 0 {
		if info, err := os.Stat(e.path); err == nil {
			return info.IsDir()
		}
	}
	return false
}

// humanReadable formats size using binary units.
func humanReadable(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}

	base := 1024.0
	units := []string{"K", "M", "G", "T", "P"}
	v := float64(size)

	for _, u := range units {
		v /= base
		if v < 99.95 {
			return fmt.Sprintf("%.1f%s", math.Round(v*10)/10, u)
		}
		if v < base-0.5 {
			return fmt.Sprintf("%.0f%s", math.Round(v), u)
		}
	}

	return "+999" + units[len(units)-1]
}

// formatTime formats t according to the active time format options.
func formatTime(t time.Time) string {
	if t.Year() == currYear {
		return t.Format(opt.timeFmtNew)
	}
	return t.Format(opt.timeFmtOld)
}

// tildePath abbreviates an absolute path under the home directory using "~".
func tildePath(path string) string {
	switch {
	case homeDir == "" || !filepath.IsAbs(path):
		return path
	case path == homeDir:
		return "~"
	default:
		if after, ok := strings.CutPrefix(path, homeDir); ok {
			return "~" + after
		}
		return path
	}
}

// showError prints e to stderr, prefixed by the program name.
func showError(e error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", progName, e)
}
