// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/tui"
	"github.com/bitwise-media-group/dotty/internal/worktree"
)

var worktreeEndCmd = &cobra.Command{
	Use:   "end [path]",
	Short: "Remove a worktree, its agent/* branch, and its tmux session.",
	Long: `Tear a worktree down: kill the tmux session named after it, remove the
worktree from its repository, and delete its branch — but only an agent/*
branch, so a manually checked-out branch survives. Uncommitted or unpushed
work is reported first; a worktree that is already gone exits quietly.

path falls back to the WorktreeRemove hook JSON on stdin
({"worktree_path": ...}).`,
	Example: `  dotty worktree end ~/Repos/dotty/.worktrees/dotty-fix-1
  echo '{"worktree_path":"..."}' | dotty worktree end   # hook form`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()

		path := argAt(args, 0)
		if path == "" {
			if data, ok := readIfPiped(cmd.InOrStdin()); ok {
				path = worktree.ParseEndPath(data)
			}
		}
		if path == "" {
			return errors.New(`no worktree path; pass it as an argument or pipe {"worktree_path": ...}`)
		}
		path, err := cli.ExpandHome(path)
		if err != nil {
			return err
		}
		return endWorktree(cmd.Context(), ios, path)
	},
}

func init() {
	worktreeCmd.AddCommand(worktreeEndCmd)
}

// endWorktree removes the worktree at path along with its tmux session and
// agent/* branch. Cleanup steps are best-effort once the path is confirmed —
// a half-removed worktree should keep getting cleaner, not error out.
func endWorktree(ctx context.Context, ios cli.IOStreams, path string) error {
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		tui.Warnf(ios, "Worktree at %s no longer exists", path)
		return nil
	}
	runner := newRunner(ios)

	branch := gitOut(ctx, runner, "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	main := mainRepo(ctx, runner, path)

	status := gitOut(ctx, runner, "-C", path, "status", "--porcelain")
	uncommitted := 0
	if status != "" {
		uncommitted = strings.Count(status, "\n") + 1
	}
	unpushed := gitOut(ctx, runner, "-C", path, "rev-list", "--count", "HEAD", "--not", "--remotes")
	if uncommitted != 0 || (unpushed != "" && unpushed != "0") {
		tui.Warnf(ios, "%s [%s] — uncommitted: %d, unpushed: %s", path, branch, uncommitted, unpushed)
	}

	session := filepath.Base(path)
	if _, err := runner.LookPath("tmux"); err == nil {
		_, _ = runner.Output(ctx, "tmux", "kill-session", "-t", session)
	}

	gitDir := []string{"-C", path}
	if main != "" {
		gitDir = []string{"-C", main}
	}
	_, _ = runner.Output(ctx, "git", append(gitDir, "worktree", "remove", "--force", path)...)
	if worktree.IsAgentBranch(branch) {
		_, _ = runner.Output(ctx, "git", append(gitDir, "branch", "-D", branch)...)
	}

	tui.Successf(ios, "Removed %s", session)
	return nil
}

// mainRepo resolves the main repository from a linked worktree (the dirname
// of its --git-common-dir).
func mainRepo(ctx context.Context, r *cli.ExecRunner, path string) string {
	common := gitOut(ctx, r, "-C", path, "rev-parse", "--git-common-dir")
	if common == "" {
		return ""
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(path, common)
	}
	return filepath.Dir(common)
}

// gitOut runs a git query and returns its trimmed output, empty on error.
func gitOut(ctx context.Context, r *cli.ExecRunner, args ...string) string {
	out, err := r.Output(ctx, "git", args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
