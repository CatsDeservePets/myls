//go:build !windows

package main

import (
	"os"
)

// mode returns an ls-style string representation for the file info.
// See https://github.com/golang/go/issues/27452 why we avoid FileMode.String
// and https://man.freebsd.org/cgi/man.cgi?ls for references.
func mode(e entry) string {
	m := e.info.Mode()
	b := []byte(m.Perm().String())
	switch {
	case m&os.ModeDevice != 0:
		if m&os.ModeCharDevice != 0 {
			b[0] = 'c'
		} else {
			b[0] = 'b'
		}
	case m&os.ModeDir != 0:
		b[0] = 'd'
	case m&os.ModeSymlink != 0:
		b[0] = 'l'
	case m&os.ModeNamedPipe != 0:
		b[0] = 'p'
	case m&os.ModeSocket != 0:
		b[0] = 's'
	default:
		b[0] = '-'
	}
	// patch exec slots with suid/sgid/sticky flags
	if m&os.ModeSetuid != 0 {
		if b[3] == 'x' {
			b[3] = 's'
		} else {
			b[3] = 'S'
		}
	}
	if m&os.ModeSetgid != 0 {
		if b[6] == 'x' {
			b[6] = 's'
		} else {
			b[6] = 'S'
		}
	}
	if m&os.ModeSticky != 0 {
		if b[9] == 'x' {
			b[9] = 't'
		} else {
			b[9] = 'T'
		}
	}

	return string(b)
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
	case m&os.ModeSocket != 0:
		return '='
	case m&0o111 != 0:
		return '*'
	default:
		return 0
	}
}
