package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"

	"golang.org/x/term"
)

// usageLine is the synopsis printed on flag parse errors.
const usageLine = `usage: myls [-h] [-V] [-a] [-d] [-l] [-r] [-1] [-color WHEN]
            [-dirsfirst] [-git] [-sort WORD] [file ...]`

// helpMessage is the full help text printed for -h/-help.
const helpMessage = `
myls - My interpretation of the ls(1) command

positional arguments:
  file          files or directories to display

options:
  -h, -help     show this help message and exit
  -V, -version  show program's version number and exit
  -a            do not ignore entries starting with .
  -d            list directories themselves, not their contents
  -l            use a long listing format
  -r            reverse order while sorting
  -1            display one entry per line
  -color WHEN   one of: always, auto, never (default: auto)
  -dirsfirst    show directories above regular files
  -git          display git status
  -sort WORD    one of: name, extension, size, time, git (default: name)

environment:
  MYLS_TIMEFMT_OLD, MYLS_TIMEFMT_NEW
                used to specify the time format for non-recent and recent files
  MYLS_DIRS_FIRST
                if set to a true boolean value, enables -dirsfirst by default
  MYLS_GIT      if set to a true boolean value, enables -git by default
  LS_COLORS     used to specify the colours for file types and file names
  NO_COLOR      if set to a non-empty value, disables coloured output by
                default; -color takes precedence`

// A sortKey specifies the primary attribute used to order entries.
type sortKey byte

const (
	name sortKey = iota
	extension
	size
	mtime
	gitStatus
	// TODO: Natural sorting
)

// Set implements the [flag.Value] interface.
func (s *sortKey) Set(value string) error {
	switch value {
	case "name":
		*s = name
	case "ext", "extension":
		*s = extension
	case "size":
		*s = size
	case "time", "mtime":
		*s = mtime
	case "git":
		*s = gitStatus
	default:
		return errors.New("must be name, extension, size, time, or git")
	}
	return nil
}

// String implements the [flag.Value] interface.
func (s sortKey) String() string {
	switch s {
	case name:
		return "name"
	case extension:
		return "extension"
	case size:
		return "size"
	case mtime:
		return "time"
	case gitStatus:
		return "git"
	default:
		return fmt.Sprintf("sortKey(%d)", s)
	}
}

// A colorMode specifies when coloured output is used.
type colorMode byte

const (
	colorAuto colorMode = iota
	colorAlways
	colorNever
)

// Set implements the [flag.Value] interface.
func (m *colorMode) Set(value string) error {
	// Secretly accept GNU aliases.
	switch value {
	case "always", "yes", "force":
		*m = colorAlways
	case "auto", "tty", "if-tty":
		*m = colorAuto
	case "never", "no", "none":
		*m = colorNever
	default:
		return errors.New("must be always, auto, or never")
	}
	return nil
}

// String implements the [flag.Value] interface.
func (m colorMode) String() string {
	switch m {
	case colorAlways:
		return "always"
	case colorAuto:
		return "auto"
	case colorNever:
		return "never"
	default:
		return fmt.Sprintf("colorMode(%d)", m)
	}
}

// options represents the program's runtime configuration.
type options struct {
	help      bool      // -h, -help
	version   bool      // -V, -version
	all       bool      // -a
	dir       bool      // -d
	long      bool      // -l
	reverse   bool      // -r
	oneEntry  bool      // -1
	color     colorMode // -color
	dirsFirst bool      // -dirsfirst
	git       bool      // -git
	sort      sortKey   // -sort
	args      []string  // non-flag command-line arguments

	timeFmtOld string
	timeFmtNew string
	termWidth  int
}

var opt options

// initOptions initializes opt from environment variables and command-line flags.
// It also handles -h/-help and -V/-version by printing a message and exiting.
func initOptions() {
	opt.timeFmtOld = cmp.Or(os.Getenv("MYLS_TIMEFMT_OLD"), "Jan _2  2006")
	opt.timeFmtNew = cmp.Or(os.Getenv("MYLS_TIMEFMT_NEW"), "Jan _2 15:04")
	opt.dirsFirst, _ = strconv.ParseBool(os.Getenv("MYLS_DIRS_FIRST"))
	opt.git, _ = strconv.ParseBool(os.Getenv("MYLS_GIT"))
	if os.Getenv("NO_COLOR") != "" {
		opt.color = colorNever // Otherwise, auto.
	}
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	opt.termWidth = cmp.Or(width, 80) // Fallback for non-terminal output etc.

	flag.BoolVar(&opt.help, "h", false, "")
	flag.BoolVar(&opt.help, "help", false, "")
	flag.BoolVar(&opt.version, "V", false, "")
	flag.BoolVar(&opt.version, "version", false, "")
	flag.BoolVar(&opt.all, "a", false, "")
	flag.BoolVar(&opt.dir, "d", false, "")
	flag.BoolVar(&opt.long, "l", false, "")
	flag.BoolVar(&opt.reverse, "r", false, "")
	flag.BoolVar(&opt.oneEntry, "1", false, "")
	flag.BoolVar(&opt.dirsFirst, "dirsfirst", opt.dirsFirst, "")
	flag.BoolVar(&opt.git, "git", opt.git, "")
	flag.Var(&opt.color, "color", "")
	flag.Var(&opt.color, "colour", "")
	flag.Var(&opt.sort, "sort", "")

	// If flag parsing fails, print the usage synopsis to stderr.
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), usageLine)
	}
	flag.Parse()

	// If -h or -help is set, print the full help text to stdout.
	if opt.help {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		fmt.Fprintln(os.Stdout, helpMessage)
		os.Exit(0)
	}

	if opt.version {
		fmt.Println(version())
		os.Exit(0)
	}

	args := flag.Args()
	// Windows leaves glob expansion to the application.
	// In this case, us.
	if runtime.GOOS == "windows" {
		args = expandGlobs(args)
	}
	if len(args) == 0 {
		args = []string{"."}
	}
	opt.args = args
}

// version returns the program name and version string.
func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "myls unknown"
	}
	return "myls " + bi.Main.Version
}

// expandGlobs expands wildcards in args using [filepath.Glob].
// If an argument returns no matches, it is left unchanged.
func expandGlobs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, pattern := range args {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			out = append(out, matches...)
		} else {
			out = append(out, pattern)
		}
	}
	return out
}
