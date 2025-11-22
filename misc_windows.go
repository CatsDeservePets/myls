//go:debug winsymlink=0

package main

import (
	"fmt"
	"os"
	"syscall"
)

// On Windows, the header is shorter than the permission string
const permSpacer = ""

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

func drawHeader() {
	if !longFlag {
		return
	}

	fmt.Printf("%s  %s %s %s\n",
		underline("Mode"),
		underline("Size"),
		underline("Date Modified"),
		underline("Name"),
	)
}
