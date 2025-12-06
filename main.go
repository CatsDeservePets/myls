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
	"time"

	"golang.org/x/term"
)

const (
	tabWidth  = 8
	usageLine = "usage: %s [-h] [-a] [-l] [-r] [-1] [-dirsfirst] [-git] [-sort WORD] [file ...]\n"
)

const helpMessage = `
myls - My interpretation of the ls(1) command

positional arguments:
  file        files or directories to display

options:
  -h, -help   show this help message and exit
  -a          do not ignore entries starting with .
  -l          use a long listing format
  -r          reverse order while sorting
  -1          display one entry per line
  -dirsfirst  show directories above regular files
  -git        display git status
  -sort WORD  one of: name, extension, size, time, git (default: name)

environment:
  MYLS_TIMEFMT_OLD, MYLS_TIMEFMT_NEW
              used to specify the time format for non-recent and recent files
  MYLS_DIRS_FIRST
              if set, behaves like -dirsfirst
  MYLS_GIT    if set, behaves like -git
`

type entry struct {
	name      string
	path      string
	info      os.FileInfo
	gitStatus string
}

type sortBy int

const (
	name sortBy = iota
	size
	mtime
	extension
	git
)

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
	helpFlag      bool
	allFlag       bool
	longFlag      bool
	reverseFlag   bool
	oneEntryFlag  bool
	dirsFirstFlag bool
	gitFlag       bool
	sortFlag      sortBy

	timeFmtOld string
	timeFmtNew string
	termWidth  int

	dirEntries = map[string][]entry{}
	gitRepos   = map[string]map[string]string{}
	currYear   = time.Now().Year()
	homeDir, _ = os.UserHomeDir()
)

func init() {
	timeFmtOld = cmp.Or(os.Getenv("MYLS_TIMEFMT_OLD"), "Jan _2  2006")
	timeFmtNew = cmp.Or(os.Getenv("MYLS_TIMEFMT_NEW"), "Jan _2 15:04")
	_, dirsFirstFlag = os.LookupEnv("MYLS_DIRS_FIRST")
	_, gitFlag = os.LookupEnv("MYLS_GIT")
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	termWidth = cmp.Or(width, 80)
}

func main() {
	flag.BoolVar(&helpFlag, "h", false, "")
	flag.BoolVar(&helpFlag, "help", false, "")
	flag.BoolVar(&allFlag, "a", false, "")
	flag.BoolVar(&longFlag, "l", false, "")
	flag.BoolVar(&reverseFlag, "r", false, "")
	flag.BoolVar(&oneEntryFlag, "1", false, "")
	flag.BoolVar(&dirsFirstFlag, "dirsfirst", dirsFirstFlag, "")
	flag.BoolVar(&gitFlag, "git", gitFlag, "")
	flag.Var(&sortFlag, "sort", "")
	flag.Usage = func() {
		// When triggered by an error, print compact version to stderr.
		fmt.Fprintf(flag.CommandLine.Output(), usageLine, os.Args[0])
	}
	flag.Parse()

	if helpFlag {
		// When user-initiated, print detailed usage message to stdout.
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		fmt.Fprint(os.Stdout, helpMessage)
		os.Exit(0)
	}

	files, dirs := collectEntries(flag.Args())
	if len(dirs) == 0 && len(files) == 0 {
		os.Exit(1)
	}
	hasOutput := len(files) > 0
	showDirName := len(files) > 0 || len(dirs) > 1

	if longFlag && gitFlag {
		attachGit(files)
	}
	sortEntries(files)
	printEntries(files)

	sortEntries(dirs)
	for _, d := range dirs {
		if hasOutput {
			// Separate directory listing from previous output.
			fmt.Println()
		}
		if showDirName {
			// If output has multiple sections, label directory
			// using the user-supplied path (abbreviated with ~).
			fmt.Printf("%s:\n", tildePath(d.name))
		}

		ents, err := readDir(d.path)
		if err != nil {
			showError(err)
		}
		if longFlag && gitFlag {
			attachGit(ents)
		}
		if allFlag {
			ents = append(selfAndParent(d.path), ents...)
		} else {
			ents = slices.DeleteFunc(ents, isHidden)
		}
		sortEntries(ents)
		printEntries(ents)
		hasOutput = true
	}
}

func collectEntries(args []string) (files, dirs []entry) {
	if len(args) == 0 {
		args = []string{"."}
	}

	for _, pattern := range args {
		paths := []string{pattern}
		// Windows does not expand shell globs automatically
		if matches, err := filepath.Glob(pattern); err == nil && len(matches) > 0 {
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
			if info.IsDir() {
				// Prefer entry type over string to simplify sorting.
				dirs = append(dirs, ent)
			} else {
				files = append(files, ent)
			}
		}
	}
	return files, dirs
}

func sortEntries(ents []entry) {
	// Always sort by name first.
	slices.SortFunc(ents, func(a, b entry) int {
		if reverseFlag {
			return strings.Compare(strings.ToLower(b.name), strings.ToLower(a.name))
		}
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})

	switch sortFlag {
	case extension:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if reverseFlag {
				return strings.Compare(strings.ToLower(filepath.Ext(b.name)), strings.ToLower(filepath.Ext(a.name)))
			}
			return strings.Compare(strings.ToLower(filepath.Ext(a.name)), strings.ToLower(filepath.Ext(b.name)))
		})
	case size:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if reverseFlag {
				return cmp.Compare(b.info.Size(), a.info.Size())
			}
			return cmp.Compare(a.info.Size(), b.info.Size())
		})
	case mtime:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if reverseFlag {
				return b.info.ModTime().Compare(a.info.ModTime())
			}
			return a.info.ModTime().Compare(b.info.ModTime())
		})
	case git:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if reverseFlag {
				return strings.Compare(strings.ToLower(b.gitStatus), strings.ToLower(a.gitStatus))
			}
			return strings.Compare(strings.ToLower(a.gitStatus), strings.ToLower(b.gitStatus))
		})
	}

	if dirsFirstFlag {
		slices.SortStableFunc(ents, func(a, b entry) int {
			if a.info.IsDir() == b.info.IsDir() {
				return 0
			}
			if a.info.IsDir() {
				return -1
			}
			return 1
		})
	}
}

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

func readDir(path string) ([]entry, error) {
	clean := filepath.Clean(path)
	if ents, ok := dirEntries[clean]; ok {
		return ents, nil
	}

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
	dirEntries[clean] = ents

	return ents, nil
}

func attachGit(ents []entry) {
	dirCache := make(map[string]map[string]string)
	for i := range ents {
		dir := filepath.Dir(ents[i].path)
		stats, ok := dirCache[dir]
		if !ok {
			stats = gitStatusesForDir(dir)
			dirCache[dir] = stats
		}
		if stats == nil {
			continue
		}
		if signs, ok := stats[ents[i].path]; ok {
			ents[i].gitStatus = strings.ReplaceAll(signs, " ", "-")
		}
	}
}

func gitStatusesForDir(dir string) map[string]string {
	root := gitRoot(dir)
	if root == "" {
		return nil
	}
	if st, ok := gitRepos[root]; ok {
		return st
	}

	cmd := exec.Command(
		"git", "-C", root,
		"status", "--porcelain=v1", "-z", "--ignored",
	)
	out, err := cmd.Output()
	if err != nil {
		gitRepos[root] = nil
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

		stats[full] = signs
	}
	gitRepos[root] = stats

	return stats
}

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

func printEntries(ents []entry) {
	if len(ents) == 0 {
		return
	}
	switch {
	case longFlag:
		printLong(ents)
	case oneEntryFlag:
		print1PerLine(ents)
	default:
		printShort(ents)
	}
}

type row struct {
	modeStr string
	sizeStr string
	timeStr string
	gitStr  string
	nameStr string
}

func printLong(ents []entry) {
	rows := make([]row, 0, len(ents))

	sizeWidth := 0
	timeWidth := 0
	gitWidth := 0

	for _, e := range ents {
		name := e.name
		if suffix := classify(e); suffix != 0 {
			name += string(suffix)
			if suffix == '@' {
				target, _ := os.Readlink(e.path)
				name += " -> " + target
			}
		}

		var sizeStr string
		if e.info.IsDir() {
			if children, err := readDir(e.path); err == nil {
				sizeStr = fmt.Sprintf("%d", len(children))
			} else {
				sizeStr = "!"
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

		gitStr := e.gitStatus
		if n := len(gitStr); n > gitWidth {
			gitWidth = n
		}

		rows = append(rows, row{
			modeStr: mode(e),
			sizeStr: sizeStr,
			timeStr: timeStr,
			gitStr:  gitStr,
			nameStr: name,
		})
	}

	for _, r := range rows {
		if gitWidth > 0 {
			if r.gitStr == "" {
				r.gitStr = "--"
			}
			fmt.Printf("%s %*s %-*s %-*s %s\n",
				r.modeStr,
				sizeWidth, r.sizeStr,
				timeWidth, r.timeStr,
				gitWidth, r.gitStr,
				r.nameStr,
			)
		} else {
			fmt.Printf("%s %*s %-*s %s\n",
				r.modeStr,
				sizeWidth, r.sizeStr,
				timeWidth, r.timeStr,
				r.nameStr,
			)
		}
	}
}

func print1PerLine(ents []entry) {
	for _, e := range ents {
		name := e.name
		if suffix := classify(e); suffix != 0 {
			name += string(suffix)
		}
		fmt.Println(name)
	}
}

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
	cols := min(max(termWidth/(colTabs*tabWidth), 1), entryCount)

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

func formatTime(t time.Time) string {
	if t.Year() == currYear {
		return t.Format(timeFmtNew)
	}
	return t.Format(timeFmtOld)
}

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

func showError(e error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], e)
}
