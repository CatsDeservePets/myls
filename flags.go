package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"golang.org/x/term"
)

const usageLine = `usage: %s [-h] [-V] [-a] [-d] [-l] [-r] [-1] [-dirsfirst] [-git]
            [-sort WORD] [file ...]
`

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
                if set, behaves like -dirsfirst
  MYLS_GIT      if set, behaves like -git
`

var (
	helpFlag      bool
	versionFlag   bool
	allFlag       bool
	dirFlag       bool
	longFlag      bool
	reverseFlag   bool
	oneEntryFlag  bool
	dirsFirstFlag bool
	gitFlag       bool
	sortFlag      sortBy

	timeFmtOld string
	timeFmtNew string
	termWidth  int
)

func parseFlags() {
	timeFmtOld = cmp.Or(os.Getenv("MYLS_TIMEFMT_OLD"), "Jan _2  2006")
	timeFmtNew = cmp.Or(os.Getenv("MYLS_TIMEFMT_NEW"), "Jan _2 15:04")
	_, dirsFirstFlag = os.LookupEnv("MYLS_DIRS_FIRST")
	_, gitFlag = os.LookupEnv("MYLS_GIT")
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	termWidth = cmp.Or(width, 80) // Fallback for non-terminal output etc.

	flag.BoolVar(&helpFlag, "h", false, "")
	flag.BoolVar(&helpFlag, "help", false, "")
	flag.BoolVar(&versionFlag, "V", false, "")
	flag.BoolVar(&versionFlag, "version", false, "")
	flag.BoolVar(&allFlag, "a", false, "")
	flag.BoolVar(&dirFlag, "d", false, "")
	flag.BoolVar(&longFlag, "l", false, "")
	flag.BoolVar(&reverseFlag, "r", false, "")
	flag.BoolVar(&oneEntryFlag, "1", false, "")
	flag.BoolVar(&dirsFirstFlag, "dirsfirst", dirsFirstFlag, "")
	flag.BoolVar(&gitFlag, "git", gitFlag, "")
	flag.Var(&sortFlag, "sort", "")
	flag.Usage = func() {
		// When triggered by an error, print compact version to stderr.
		fmt.Fprintf(flag.CommandLine.Output(), usageLine, progName)
	}
	flag.Parse()

	if helpFlag {
		// When user-initiated, print detailed usage message to stdout.
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		fmt.Fprint(os.Stdout, helpMessage)
		os.Exit(0)
	}
	if versionFlag {
		fmt.Println(version())
		os.Exit(0)
	}
}

func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return progName + " unknown"
	}
	return fmt.Sprintf("%s %s", progName, bi.Main.Version)
}
