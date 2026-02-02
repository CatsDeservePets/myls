//go:debug winsymlink=0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// execExts is the set of executable filename extensions on Windows.
var execExts = map[string]bool{}

func init() {
	s := os.Getenv("PATHEXT")
	if s == "" {
		s = ".COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC;.CPL"
	}
	for ext := range strings.SplitSeq(s, ";") {
		if ext == "" {
			continue
		}
		execExts[strings.ToLower(ext)] = true
	}
}

// mode returns a PowerShell-style file mode string for e.
// See https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.management/get-childitem
// and https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.management/get-item
// for reference.
// Note: Some examples still show the obsolete 6-character mode format.
func mode(e entry) string {
	m := e.info.Mode()
	b := [5]byte{'-', '-', '-', '-', '-'}
	switch {
	case m&os.ModeSymlink != 0:
		b[0] = 'l'
	case e.info.IsDir():
		b[0] = 'd'
	}

	if sys, ok := e.info.Sys().(*syscall.Win32FileAttributeData); ok && sys != nil {
		attrs := sys.FileAttributes

		if attrs&syscall.FILE_ATTRIBUTE_ARCHIVE != 0 {
			b[1] = 'a'
		}
		if attrs&syscall.FILE_ATTRIBUTE_READONLY != 0 {
			b[2] = 'r'
		}
		if attrs&syscall.FILE_ATTRIBUTE_HIDDEN != 0 {
			b[3] = 'h'
		}
		if attrs&syscall.FILE_ATTRIBUTE_SYSTEM != 0 {
			b[4] = 's'
		}
	}

	return string(b[:])
}

// classify returns an ls-style type indicator for e, or 0 if none applies.
func classify(e entry) rune {
	m := e.info.Mode()
	switch {
	case m&os.ModeSymlink != 0:
		return '@'
	case m&os.ModeDir != 0:
		return os.PathSeparator
	case m&os.ModeNamedPipe != 0:
		return '|'
	case isExecutable(e):
		return '*'
	default:
		return 0
	}
}

// isExecutable reports whether e should be treated as executable.
// This is determined by comparing the file extension against %PATHEXT%.
func isExecutable(e entry) bool {
	if e.info.IsDir() {
		return false
	}
	_, ok := execExts[strings.ToLower(filepath.Ext(e.uiName))]
	return ok
}

// isHidden reports whether e's name begins with a dot or has the hidden
// attribute set.
func isHidden(e entry) bool {
	hidden := strings.HasPrefix(e.uiName, ".")
	if !hidden {
		if sys, ok := e.info.Sys().(*syscall.Win32FileAttributeData); ok && sys != nil {
			hidden = sys.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
		}
	}
	return hidden
}
