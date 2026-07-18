// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/profile"
	"github.com/bitwise-media-group/dotty/internal/scaffold"
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage agent git worktrees.",
	Long: `The agent worktree lifecycle: start creates or reuses a worktree on an
agent/<name> branch and prints its path; end removes the worktree, its tmux
session, and the agent/* branch. Both verbs also accept their agent hook
JSON on stdin (WorktreeCreate and WorktreeRemove), so they wire directly
into claude's hooks.

Worktrees live at the location the active profile configures (dotty init
--worktrees): a directory inside each repository — the default .worktrees,
kept out of git by the shared ignore — or one absolute shared root. Inside
them, commit and tag signing is off (sandboxed agents cannot reach the
security key); re-sign afterwards with dotty git resign.`,
	Example: `  dotty worktree start myrepo fix-tests
  dotty worktree end`,
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
}

// worktreesSetting resolves the configured worktree location: the exported
// DOTTY_WORKTREES (present in any shell the profile env reached), the active
// profile's answers, or the default.
func worktreesSetting() string {
	if v := os.Getenv("DOTTY_WORKTREES"); v != "" {
		return v
	}
	if configDir, err := cli.ConfigDir(); err == nil {
		if activeDir, err := profile.ActiveDir(configDir); err == nil {
			if answers, err := scaffold.LoadAnswers(activeDir); err == nil && answers.Worktrees != "" {
				return answers.Worktrees
			}
		}
	}
	return scaffold.DefaultWorktrees
}

// readIfPiped returns stdin's contents when it is not a terminal (an agent
// hook pipe). For a tty it returns nothing — a bare interactive invocation
// must not hang waiting for input.
func readIfPiped(in io.Reader) ([]byte, bool) {
	if f, ok := in.(*os.File); ok && cli.IsTerminal(f) {
		return nil, false
	}
	data, err := io.ReadAll(in)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}
