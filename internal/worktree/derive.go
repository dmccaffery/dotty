// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package worktree

import (
	"path/filepath"
	"strings"
)

// Names are the identifiers derived for a worktree from a repo path and
// suffix. Name doubles as the tmux session name, so it carries the repo
// basename even when the worktree lives inside the repo.
type Names struct {
	Name   string // sanitized "<repo-basename>-<suffix>"
	Branch string // "agent/<name>"
	Path   string // "<root>/<name>"
}

// Root resolves the configured worktree location against a repository: a
// relative setting (the default .worktrees) is a directory inside the repo,
// an absolute or ~-prefixed one is a shared root.
func Root(repo, setting, home string) string {
	switch {
	case setting == "":
		return filepath.Join(repo, ".worktrees")
	case strings.HasPrefix(setting, "~/"):
		return filepath.Join(home, setting[2:])
	case filepath.IsAbs(setting):
		return setting
	default:
		return filepath.Join(repo, setting)
	}
}

// Derive computes the worktree name, branch, and path under root.
func Derive(repoPath, suffix, root string) Names {
	name := Sanitize(filepath.Base(repoPath) + "-" + suffix)
	return Names{
		Name:   name,
		Branch: "agent/" + name,
		Path:   filepath.Join(root, name),
	}
}

// IsAgentBranch reports whether branch is an agent/* branch — the guard that
// stops `worktree end` from deleting a non-agent branch.
func IsAgentBranch(branch string) bool {
	return strings.HasPrefix(branch, "agent/")
}
