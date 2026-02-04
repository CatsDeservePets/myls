package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	gitRepos   = map[string]map[string]string{}
	gitReposMu sync.Mutex
)

// attachGitToFiles populates gitStatus for ents, doing at most one lookup
// per directory.
func attachGitToFiles(ents []entry) {
	dirCache := make(map[string]map[string]string, len(ents))
	showGit := false

	for i := range ents {
		e := &ents[i]
		var dir string
		if e.info.IsDir() {
			// For directory entries, use directory itself; it may be the repo root.
			dir = e.fullPath
		} else {
			dir = filepath.Dir(e.fullPath)
		}

		stats, ok := dirCache[dir]
		if !ok {
			stats = gitStatusesForDir(dir)
			dirCache[dir] = stats
		}
		if stats == nil {
			continue
		}
		showGit = true
		if signs, ok := stats[e.fullPath]; ok {
			e.gitStatus = strings.ReplaceAll(signs, " ", "-")
		}
	}

	// Only add placeholders if any Git status is present.
	if !showGit {
		return
	}
	for i := range ents {
		if ents[i].gitStatus == "" {
			ents[i].gitStatus = "--"
		}
	}
}

// attachGitToDir populates gitStatus for ents using dir's repository.
func attachGitToDir(dir string, ents []entry) {
	stats := gitStatusesForDir(dir)
	if stats == nil {
		return
	}

	for i := range ents {
		e := &ents[i]
		if signs, ok := stats[e.fullPath]; ok {
			e.gitStatus = strings.ReplaceAll(signs, " ", "-")
		} else {
			e.gitStatus = "--"
		}
	}
}

// gitPriority ranks Git status codes by significance (higher wins).
func gitPriority(signs string) int {
	switch signs {
	case "!!":
		return 1
	case "??":
		return 2
	default:
		return 3
	}
}

// gitStatusesForDir returns Git status codes for dir's repository.
// The map keys are absolute paths for all entries reported by Git.
// It returns nil if no repository is found or status cannot be read.
func gitStatusesForDir(dir string) map[string]string {
	root := gitRoot(dir)
	if root == "" {
		return nil
	}

	gitReposMu.Lock()
	if st, ok := gitRepos[root]; ok {
		gitReposMu.Unlock()
		return st
	}
	gitReposMu.Unlock()

	cmd := exec.Command(
		"git",
		"-C", root,
		"status",
		"--porcelain=v1",
		"-z",
		"--ignored=matching",
	)
	out, err := cmd.Output()
	if err != nil {
		gitReposMu.Lock()
		gitRepos[root] = nil
		gitReposMu.Unlock()
		return nil
	}

	stats := make(map[string]string)
	for rec := range bytes.SplitSeq(out, []byte{0}) {
		// skip invalid status (e.g. second part of rename entry)
		if len(rec) < 4 || rec[2] != ' ' {
			continue
		}
		signs := string(rec[:2])
		rel := string(rec[3:])
		rel = filepath.FromSlash(rel)
		full := filepath.Join(root, rel)

		if prev, ok := stats[full]; !ok || gitPriority(prev) < gitPriority(signs) {
			stats[full] = signs
		}

		// propagate "highest" status to all parent dirs
		dirPath := filepath.Dir(full)
		for len(dirPath) >= len(root) {
			prev, ok := stats[dirPath]
			if !ok || gitPriority(prev) < gitPriority(signs) {
				stats[dirPath] = signs
			}
			if dirPath == root {
				break
			}
			parent := filepath.Dir(dirPath)
			if parent == dirPath {
				break
			}
			dirPath = parent
		}
	}

	gitReposMu.Lock()
	gitRepos[root] = stats
	gitReposMu.Unlock()

	return stats
}

// gitRoot returns the repository root containing dir, or "" if none is found.
func gitRoot(dir string) string {
	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			return ""
		}
		root = parent
	}
}
