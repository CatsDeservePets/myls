package main

import (
	"cmp"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
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
	permSpacer = " "
)

func init() {
	if runtime.GOOS == "windows" {
		permSpacer = ""
	}
}

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
				fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
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
	showDirHeader := len(dirs) > 1 || len(files) > 0

	drawHeader()

	for _, e := range files {
		printEntry(e)
	}

	for _, d := range dirs {
		if hasOutput {
			fmt.Println() // Separate directory listing from previous output.
		}
		if showDirHeader {
			fmt.Printf("%s:\n", d) // Label directory when multiple sections exist.
			drawHeader()
		}

		if err := listDir(d); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		}
		hasOutput = true
	}
}

func listDir(dir string) error {
	ents, err := readDir(dir)
	if err != nil {
		return err
	}

	if allFlag {
		// Current and parent dir
		for _, name := range [...]string{".", ".."} {
			full := filepath.Join(dir, name)
			info, err := os.Lstat(full)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
				continue
			}

			printEntry(entry{name, full, info})
		}
	}

	for _, e := range ents {
		if !allFlag && strings.HasPrefix(e.name, ".") {
			continue
		}
		printEntry(e)
	}

	return nil
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
			fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
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

func printEntry(e entry) {
	name := e.name

	if suffix := classify(e); suffix != 0 {
		name += string(suffix)
		if suffix == '@' && longFlag {
			target, _ := os.Readlink(e.path)
			name += " -> " + target
		}
	}

	if longFlag {
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
		fmt.Printf("%s%s%5s %s %s\n",
			mode(e),
			permSpacer,
			size,
			formatTime(e.info.ModTime()),
			name,
		)
	} else {
		fmt.Println(name)
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
		return t.Format("Jan _2 15:04 ")
	}
	return t.Format("Jan _2  2006 ")
}

func drawHeader() {
	if !longFlag {
		return
	}

	if runtime.GOOS == "windows" {
		fmt.Printf("%s  %s %s %s\n",
			underline("Mode"),
			underline("Size"),
			underline("Date Modified"),
			underline("Name"),
		)
	} else {
		fmt.Printf("%s %s %s %s\n",
			underline("Permissions"),
			underline("Size"),
			underline("Date Modified"),
			underline("Name"),
		)
	}
}

func underline(s string) string {
	// underline + string + reset
	return "\033[4m" + s + "\033[0m"
}
