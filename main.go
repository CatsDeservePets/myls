package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type fileEntry struct {
	path string
	info os.FileInfo
}

var (
	helpFlag bool
	allFlag  bool
	longFlag bool
)

var currYear = time.Now().Year()

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
	var files []fileEntry

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
				files = append(files, fileEntry{path: p, info: info})
			}
		}
	}

	if len(dirs) == 0 && len(files) == 0 {
		os.Exit(1)
	}

	for _, f := range files {
		printEntry(f.path, f.info)
	}

	hasOutput := len(files) > 0
	showDirHeader := len(dirs) > 1 || len(files) > 0

	for _, d := range dirs {
		if hasOutput {
			fmt.Println() // Separate directory listing from previous output.
		}
		if showDirHeader {
			fmt.Printf("%s:\n", d) // Label directory when multiple sections exist.
		}
		if err := listDir(d); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		}
		hasOutput = true
	}
}

func listDir(dir string) error {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	if allFlag {
		// Current and parent dir
		for _, e := range [...]string{".", ".."} {
			info, err := os.Lstat(filepath.Join(dir, e))
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
				continue
			}

			printEntry(e, info)
		}
	}

	for _, e := range ents {
		name := e.Name()
		if !allFlag && strings.HasPrefix(name, ".") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
			continue
		}

		printEntry(name, info)
	}

	return nil
}

func printEntry(s string, info os.FileInfo) {
	if suffix := classify(info.Mode()); suffix != 0 {
		s += string(suffix)
	}
	if longFlag {
		// TODO: calculate alignment
		fmt.Printf("%s %8d %s %s\n",
			info.Mode().String(),
			info.Size(),
			formatTime(info.ModTime()),
			s,
		)
	} else {
		fmt.Println(s)
	}
}

func classify(m os.FileMode) rune {
	switch {
	case m&os.ModeSymlink != 0:
		return '@'
	case m.IsDir():
		return os.PathSeparator
	case m&os.ModeNamedPipe != 0:
		return '|'
	case m&os.ModeSocket != 0:
		return '='
	case m&0o111 != 0:
		return '*'
	default:
		return 0
	}
}

func formatTime(t time.Time) string {
	if t.Year() == currYear {
		return t.Format("Jan _2 15:04")
	}
	return t.Format("Jan _2  2006")
}
