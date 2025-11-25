package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type entry struct {
	name string
	path string
	info os.FileInfo
}

type sortBy int

const (
	name sortBy = iota
	size
	mtime
	extension
)

func (s *sortBy) Set(cmp string) error {
	switch cmp {
	case "name":
		*s = name
	case "ext", "extension":
		*s = extension
	case "size":
		*s = size
	case "time", "mtime":
		*s = mtime
	default:
		return errors.New("must be name, extension, size, or time")
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
	default:
		return ""
	}
}

var (
	helpFlag    bool
	allFlag     bool
	longFlag    bool
	reverseFlag bool
	sortFlag    sortBy
)

var (
	dirEntries = map[string][]entry{}
	currYear   = time.Now().Year()
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
  -sort WORD  one of: name, extension, size, time (default: name)
`

func main() {
	flag.BoolVar(&helpFlag, "h", false, "")
	flag.BoolVar(&helpFlag, "help", false, "")
	flag.BoolVar(&allFlag, "a", false, "")
	flag.BoolVar(&longFlag, "l", false, "")
	flag.BoolVar(&reverseFlag, "r", false, "")
	flag.Var(&sortFlag, "sort", "")
	flag.Usage = func() {
		// When triggered by an error, print compact version to stderr.
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [-h] [-a] [-l] [-r] [-sort WORD] [file ...]\n", os.Args[0])
	}
	flag.Parse()

	if helpFlag {
		// When user-initiated, print detailed usage message to stdout.
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		fmt.Fprint(os.Stdout, helpMessage)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	var files, dirs []entry

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

			ent := entry{p, p, info}
			if info.IsDir() {
				// Prefer entry type over string to simplify sorting.
				dirs = append(dirs, ent)
			} else {
				files = append(files, ent)
			}
		}
	}

	if len(dirs) == 0 && len(files) == 0 {
		os.Exit(1)
	}
	hasOutput := len(files) > 0
	showDirName := len(files) > 0 || len(dirs) > 1

	sort(files)
	sort(dirs)

	printEntries(files)

	for _, d := range dirs {
		if hasOutput {
			fmt.Println() // Separate directory listing from previous output.
		}
		if showDirName {
			fmt.Printf("%s:\n", d.name) // Label directory when multiple sections exist.
		}

		ents, err := readDir(d.name)
		sort(ents)
		if err != nil {
			showError(err)
		}
		if allFlag {
			ents = append(selfAndParent(d.name), ents...)
		} else {
			ents = slices.DeleteFunc(ents, isHidden)
		}
		printEntries(ents)
		hasOutput = true
	}
}

func sort(ents []entry) {
	// Always sort by name first
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
	case name:
		// We already did that, why would we do that again?
	case size:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if reverseFlag {
				return cmp.Compare(b.info.Size(), (a.info.Size()))
			}
			return cmp.Compare(a.info.Size(), (b.info.Size()))
		})
	case mtime:
		slices.SortStableFunc(ents, func(a, b entry) int {
			if reverseFlag {
				return b.info.ModTime().Compare(a.info.ModTime())
			}
			return a.info.ModTime().Compare(b.info.ModTime())
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
			ents = append(ents, entry{name, full, info})
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
		ents = append(ents, entry{name, full, info})
	}
	dirEntries[clean] = ents

	return ents, nil
}

func printEntries(ents []entry) {
	if len(ents) == 0 {
		return
	}
	if longFlag {
		printLong(ents)
	} else {
		printShort(ents)
	}
}

func printShort(ents []entry) {
	for _, e := range ents {
		name := e.name
		if suffix := classify(e); suffix != 0 {
			name += string(suffix)
		}
		fmt.Println(name)
	}
}

type row struct {
	modeStr string
	sizeStr string
	timeStr string
	nameStr string
}

func printLong(ents []entry) {
	rows := make([]row, 0, len(ents))

	sizeWidth := 0
	timeWidth := 0

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
			if ents, err := readDir(e.path); err == nil {
				sizeStr = fmt.Sprintf("%d", len(ents))
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

		rows = append(rows, row{
			modeStr: mode(e),
			sizeStr: sizeStr,
			timeStr: timeStr,
			nameStr: name,
		})
	}

	for _, r := range rows {
		fmt.Printf("%s %*s %-*s %s\n",
			r.modeStr,
			sizeWidth, r.sizeStr,
			timeWidth, r.timeStr,
			r.nameStr,
		)
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
		return t.Format("Jan _2 15:04")
	}
	return t.Format("Jan _2  2006")
}

func showError(e error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], e)
}
