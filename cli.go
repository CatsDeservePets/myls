package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"

	"golang.org/x/term"
)

// usageLine is the synopsis printed on flag parse errors.
const usageLine = `usage: %s [-h] [-V] [-a] [-d] [-l] [-r] [-1] [-dirsfirst] [-git]
            [-sort WORD] [file ...]
`

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
  -dirsfirst    show directories above regular files
  -git          display git status
  -sort WORD    one of: name, extension, size, time, git (default: name)

environment:
  MYLS_TIMEFMT_OLD, MYLS_TIMEFMT_NEW
                used to specify the time format for non-recent and recent files
  MYLS_DIRS_FIRST
                if set to a true value, enables -dirsfirst by default
  MYLS_GIT      if set to a true value, enables -git by default
`

// options represents the program's runtime configuration.
type options struct {
	help      bool   // -h, -help
	version   bool   // -V, -version
	all       bool   // -a
	dir       bool   // -d
	long      bool   // -l
	reverse   bool   // -r
	oneEntry  bool   // -1
	dirsFirst bool   // -dirsfirst
	git       bool   // -git
	sort      sortBy // -sort

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
	flag.Var(&opt.sort, "sort", "")

	// If flag parsing fails, print the usage synopsis to stderr.
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usageLine, progName)
	}
	flag.Parse()

	// If -h or -help is set, print the full help text to stdout.
	if opt.help {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		fmt.Fprint(os.Stdout, helpMessage)
		os.Exit(0)
	}

	if opt.version {
		fmt.Println(version())
		os.Exit(0)
	}
}

// version returns the program name and version string.
func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return progName + " unknown"
	}
	return fmt.Sprintf("%s %s", progName, bi.Main.Version)
}
