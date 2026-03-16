package main

import (
	"os"
	"strings"
)

const (
	csi   = "\033[" // Control Sequence Introducer
	reset = "\033[0m"
)

type colorConfig struct {
	enabled  bool
	types    map[string]string
	suffixes map[string]string
}

var colors = colorConfig{
	enabled:  false,
	suffixes: make(map[string]string),
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

func initColors() {
	if os.Getenv("NO_COLOR") != "" {
		return
	}
	if v := os.Getenv("LS_COLORS"); v != "" {
		colors.enabled = true
		parseLSCOLORS(v)
	}
}

func parseLSCOLORS(s string) {
	for ent := range strings.SplitSeq(s, ":") {
		k, v, found := strings.Cut(ent, "=")
		if !found {
			continue
		}
		// Treat reset-only styles as no-op.
		if v == "0" || v == "00" {
			v = ""
		}
		k, _ = strings.CutPrefix(k, "*")
		if _, ok := colors.types[k]; ok {
			colors.types[k] = v
		} else {
			colors.suffixes[k] = v
		}
	}
}

func colorize(e entry) string {
	if !colors.enabled {
		return e.uiName
	}

	var kind string
	m := e.info.Mode()
	switch {
	case e.linkMode == working:
		kind = "ln"
	case e.linkMode == orphan:
		kind = "or"
	case m&os.ModeDir != 0 && m&os.ModeSticky != 0 && m&0o002 != 0:
		kind = "tw"
	case m&os.ModeDir != 0 && m&0o002 != 0:
		kind = "ow"
	case m&os.ModeDir != 0 && m&os.ModeSticky != 0:
		kind = "st"
	case m&os.ModeDir != 0:
		kind = "di"
	case m&os.ModeNamedPipe != 0:
		kind = "pi"
	case m&os.ModeSocket != 0:
		kind = "so"
	case m&os.ModeCharDevice != 0:
		kind = "cd"
	case m&os.ModeDevice != 0:
		kind = "bd"
	case m&os.ModeType == 0 && m&os.ModeSetuid != 0:
		kind = "su"
	case m&os.ModeType == 0 && m&os.ModeSetgid != 0:
		kind = "sg"
	case isExecutable(e):
		kind = "ex"
	}

	if style, ok := colors.types[kind]; ok {
		return sgr(style, e.uiName)
	}

	for k, v := range colors.suffixes {
		if strings.HasSuffix(e.uiName, k) {
			return sgr(v, e.uiName)
		}
	}

	// Fall back to regular files.
	if style, ok := colors.types["fi"]; ok {
		return sgr(style, e.uiName)
	}

	return e.uiName
}

// sgr applies style to s and returns it as a valid ANSI escape sequence.
func sgr(style, s string) string {
	if style == "" {
		return s
	}
	return csi + style + "m" + s + reset
}
