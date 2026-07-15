package main

import (
	"os"
	"slices"
	"strings"

	"golang.org/x/term"
)

const (
	csi   = "\033["   // ANSI control sequence introducer
	reset = "\033[0m" // ANSI reset sequence
)

// colorConfig represents the programs's colour configuration.
type colorConfig struct {
	enabled  bool              // whether coloured output should be used
	types    map[string]string // $LS_COLORS type to colour sequence (e.g. "di", "ln")
	suffixes []suffixRule      // filename suffix to colour sequence, kept in a slice for sorting
}

type suffixRule struct {
	suffix string // filename suffix (e.g. ".go", "~")
	style  string // colour sequence
}

var colors = colorConfig{
	types: map[string]string{
		"ln": "", // LINK
		"or": "", // ORPHAN
		"tw": "", // STICKY_OTHER_WRITABLE
		"ow": "", // OTHER_WRITABLE
		"st": "", // STICKY
		"di": "", // DIR
		"pi": "", // FIFO
		"so": "", // SOCK
		"cd": "", // CHR
		"bd": "", // BLK
		"su": "", // SETUID
		"sg": "", // SETGID
		"ex": "", // EXEC
		"fi": "", // FILE

		/* not implemented */
		"no": "", // NORMAL
		"rs": "", // RESET
		"do": "", // DOOR
		"mh": "", // MULTIHARDLINK
		"mi": "", // MISSING
		"ca": "", // CAPABILITY
	},
}

// applyLSCOLORS parses an $LS_COLORS value and updates c with its rules.
// Note: BSD's $LSCOLORS uses a different format and is not supported.
func (c *colorConfig) applyLSCOLORS(s string) {
	for ent := range strings.SplitSeq(s, ":") {
		k, v, found := strings.Cut(ent, "=")
		if !found {
			continue
		}
		// Treat reset-only styles as no-op.
		if v == "0" || v == "00" {
			v = ""
		}
		if _, ok := c.types[k]; ok {
			c.types[k] = v
		} else if k, _ = strings.CutPrefix(k, "*"); k != "" {
			c.suffixes = append(c.suffixes, suffixRule{k, v})
		}
	}
	// Sort longest suffix first so the earliest match is the most specific.
	slices.SortStableFunc(c.suffixes, func(a, b suffixRule) int {
		if n := len(b.suffix) - len(a.suffix); n != 0 {
			return n
		}
		return strings.Compare(a.suffix, b.suffix)
	})
}

// initColors initialises the colour configuration from environment variables.
func initColors() {
	if os.Getenv("NO_COLOR") != "" || !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	if v := os.Getenv("LS_COLORS"); v != "" {
		colors.enabled = true
		colors.applyLSCOLORS(v)
	}
}

// colorize adds colours to e's uiName and returns it.
func colorize(e entry) string {
	if !colors.enabled {
		return e.uiName
	}

	types := colors.types
	m := e.info.Mode()
	style := ""

	setStyle := func(kind string) bool {
		style = types[kind]
		return style != ""
	}

	// Order matters: if a matching style is unset, try the next fallback.
	switch {
	case e.linkState == orphanedLink && setStyle("or"):
	case e.linkState != noLink && setStyle("ln"):

	case m&os.ModeDir != 0 && m&os.ModeSticky != 0 && m&0o002 != 0 && setStyle("tw"):
	case m&os.ModeDir != 0 && m&0o002 != 0 && setStyle("ow"):
	case m&os.ModeDir != 0 && m&os.ModeSticky != 0 && setStyle("st"):
	case m&os.ModeDir != 0 && setStyle("di"):

	case m&os.ModeNamedPipe != 0 && setStyle("pi"):
	case m&os.ModeSocket != 0 && setStyle("so"):
	case m&os.ModeCharDevice != 0 && setStyle("cd"):
	case m&os.ModeDevice != 0 && setStyle("bd"):

	case m&os.ModeType == 0 && m&os.ModeSetuid != 0 && setStyle("su"):
	case m&os.ModeType == 0 && m&os.ModeSetgid != 0 && setStyle("sg"):
	case isExecutable(e) && setStyle("ex"):
	}

	if style != "" {
		return sgr(style, e.uiName)
	}

	if m&os.ModeType == 0 {
		for _, s := range colors.suffixes {
			// TODO: should we also match against [entry.sortName]
			// to catch files with an uppercase file extension?
			if strings.HasSuffix(e.uiName, s.suffix) {
				return sgr(s.style, e.uiName)
			}
		}
	}

	// Fall back to regular files.
	return sgr(types["fi"], e.uiName)
}

// sgr applies style to s and returns it as a valid ANSI escape sequence.
func sgr(style, s string) string {
	if style == "" {
		return s
	}
	return csi + style + "m" + s + reset
}
