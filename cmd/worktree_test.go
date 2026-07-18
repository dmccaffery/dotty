// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// worktreeRepo builds a real git repository with one commit and points HOME
// and the XDG dirs at scratch space (no active profile: the default
// .worktrees applies).
func worktreeRepo(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("DOTTY_WORKTREES", "")

	repo := filepath.Join(home, "Repos", "proj")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"-c", "user.name=t", "-c", "user.email=t@x", "commit", "--allow-empty", "--no-gpg-sign", "-m", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return repo
}

func TestWorktreeStartAndEnd(t *testing.T) {
	repo := worktreeRepo(t)
	want := filepath.Join(repo, ".worktrees", "proj-fix-1")

	if err := execDotty(t, "worktree", "start", repo, "fix-1"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("worktree not created at %s: %v", want, err)
	}
	out, err := exec.Command("git", "-C", want, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(out)) != "agent/proj-fix-1" {
		t.Fatalf("worktree branch = %q, %v", out, err)
	}

	// start is idempotent: an existing worktree is reused, not an error.
	if err := execDotty(t, "worktree", "start", repo, "fix-1"); err != nil {
		t.Fatalf("re-start: %v", err)
	}

	if err := execDotty(t, "worktree", "end", want); err != nil {
		t.Fatalf("end: %v", err)
	}
	if _, err := os.Stat(want); err == nil {
		t.Fatal("worktree still present after end")
	}
	branchRef := []string{"-C", repo, "show-ref", "--verify", "--quiet", "refs/heads/agent/proj-fix-1"}
	if err := exec.Command("git", branchRef...).Run(); err == nil {
		t.Fatal("agent branch survived end")
	}
	// Ending an already-gone worktree is quiet, not an error.
	if err := execDotty(t, "worktree", "end", want); err != nil {
		t.Fatalf("re-end: %v", err)
	}
}

func TestWorktreeEndSparesNonAgentBranches(t *testing.T) {
	repo := worktreeRepo(t)
	path := filepath.Join(repo, ".worktrees", "manual")
	add := exec.Command("git", "-C", repo, "worktree", "add", "-b", "feature/manual", path)
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("worktree add: %v\n%s", err, out)
	}

	if err := execDotty(t, "worktree", "end", path); err != nil {
		t.Fatalf("end: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("worktree still present")
	}
	manualRef := []string{"-C", repo, "show-ref", "--verify", "--quiet", "refs/heads/feature/manual"}
	if err := exec.Command("git", manualRef...).Run(); err != nil {
		t.Fatal("non-agent branch was deleted")
	}
}

func TestWorktreeStartHonorsAbsoluteRoot(t *testing.T) {
	repo := worktreeRepo(t)
	root := filepath.Join(os.Getenv("HOME"), "wt-root")
	t.Setenv("DOTTY_WORKTREES", root)

	if err := execDotty(t, "worktree", "start", repo, "abs"); err != nil {
		t.Fatalf("start: %v", err)
	}
	want := filepath.Join(root, "proj-abs")
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("worktree not at absolute root %s: %v", want, err)
	}
}
