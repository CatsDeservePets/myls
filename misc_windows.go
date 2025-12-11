//go:debug winsymlink=0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

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

// mode returns a Powershell-style string representation for the file info.
// See https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.management/get-childitem
// and https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.management/get-item
// for references.
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

func isExecutable(e entry) bool {
	if e.info.IsDir() {
		return false
	}
	_, ok := execExts[strings.ToLower(filepath.Ext(e.name))]
	return ok
}

func isHidden(e entry) bool {
	hidden := strings.HasPrefix(e.name, ".")
	if !hidden {
		if sys, ok := e.info.Sys().(*syscall.Win32FileAttributeData); ok && sys != nil {
			hidden = sys.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
		}
	}
	return hidden
}
