package main

import (
	"cmp"
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

var (
	helpFlag bool
	allFlag  bool
	longFlag bool
)

var (
	dirEntries = map[string][]entry{}
	currYear   = time.Now().Year()
)

const helpMessage = `
myls - My interpretation of the ls(1) command

positional arguments:
  file       files or directories to display

options:
  -h, -help  show this help message and exit
  -a         do not ignore entries starting with .
  -l         use a long listing format
`

func main() {
	flag.BoolVar(&helpFlag, "h", false, "")
	flag.BoolVar(&helpFlag, "help", false, "")
	flag.BoolVar(&allFlag, "a", false, "")
	flag.BoolVar(&longFlag, "l", false, "")
	flag.Usage = func() {
		// When triggered by an error, print compact version to stderr.
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [-h] [-a] [-l] [file ...]\n", os.Args[0])
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

	var dirs []string
	var files []entry

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

			if info.IsDir() {
				dirs = append(dirs, p)
			} else {
				files = append(files, entry{p, p, info})
			}
		}
	}

	if len(dirs) == 0 && len(files) == 0 {
		os.Exit(1)
	}
	hasOutput := len(files) > 0
	showDirName := len(files) > 0 || len(dirs) > 1

	printEntries(files)

	for _, d := range dirs {
		if hasOutput {
			fmt.Println() // Separate directory listing from previous output.
		}
		if showDirName {
			fmt.Printf("%s:\n", d) // Label directory when multiple sections exist.
		}

		ents, err := readDir(d)
		if err != nil {
			showError(err)
		}
		if allFlag {
			ents = append(selfAndParent(d), ents...)
		} else {
			ents = slices.DeleteFunc(ents, isHidden)
		}
		printEntries(ents)
		hasOutput = true
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

	slices.SortFunc(ents, func(a, b entry) int {
		return cmp.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})
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

func printLong(ents []entry) {
	for _, e := range ents {
		name := e.name

		if suffix := classify(e); suffix != 0 {
			name += string(suffix)
			if suffix == '@' {
				target, _ := os.Readlink(e.path)
				name += " -> " + target
			}
		}

		var size string
		if e.info.IsDir() {
			if ents, err := readDir(e.path); err == nil {
				size = fmt.Sprintf("%d", len(ents))
			} else {
				size = "!"
			}
		} else {
			size = humanReadable(e.info.Size())
		}
		// TODO: calculate alignment
		fmt.Printf("%s %5s %s %s\n",
			mode(e),
			size,
			formatTime(e.info.ModTime()),
			name,
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
