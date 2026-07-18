// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package linker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// write creates path (and parents) holding content.
func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// assertLink fails unless site is a symlink pointing at dest.
func assertLink(t *testing.T, site, dest string) {
	t.Helper()
	got, err := os.Readlink(site)
	if err != nil {
		t.Fatalf("Readlink(%s): %v", site, err)
	}
	if got != dest {
		t.Fatalf("link %s points at %s, want %s", site, got, dest)
	}
}

// resolveAll returns a Resolver that always answers res.
func resolveAll(res Resolution) Resolver {
	return func(Conflict) (Resolution, error) { return res, nil }
}

func TestApplyLinksAndFolds(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, filepath.Join(source, "file"), "f")
	write(t, filepath.Join(source, "sub", "a"), "a")
	write(t, filepath.Join(source, "sub", "b"), "b")

	rep, err := Apply(Tree{Source: source, Target: target}, resolveAll(ResFail), "")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(rep.Linked) != 2 {
		t.Fatalf("Linked = %v, want file and folded sub", rep.Linked)
	}
	assertLink(t, filepath.Join(target, "file"), filepath.Join(source, "file"))
	assertLink(t, filepath.Join(target, "sub"), filepath.Join(source, "sub"))
}

func TestApplyIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, filepath.Join(source, "file"), "f")
	write(t, filepath.Join(source, "sub", "a"), "a")

	if _, err := Apply(Tree{Source: source, Target: target}, resolveAll(ResFail), ""); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	rep, err := Apply(Tree{Source: source, Target: target}, resolveAll(ResFail), "")
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if len(rep.Linked)+len(rep.Replaced)+len(rep.Backed)+len(rep.Adopted)+len(rep.Skipped) != 0 {
		t.Fatalf("second Apply changed something: %+v", rep)
	}
	if rep.OK != 2 {
		t.Fatalf("OK = %d, want 2", rep.OK)
	}
}

func TestApplyUnfoldsIntoRealDir(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, filepath.Join(source, "sub", "a"), "a")
	write(t, filepath.Join(target, "sub", "keep"), "user file already there")

	rep, err := Apply(Tree{Source: source, Target: target}, resolveAll(ResFail), "")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	assertLink(t, filepath.Join(target, "sub", "a"), filepath.Join(source, "sub", "a"))
	got, err := os.ReadFile(filepath.Join(target, "sub", "keep"))
	if err != nil || string(got) != "user file already there" {
		t.Fatalf("unrelated file disturbed: %q, %v", got, err)
	}
	if len(rep.Linked) != 1 {
		t.Fatalf("Linked = %v, want just sub/a", rep.Linked)
	}
}

func TestApplyReplacesStaleSymlinks(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, filepath.Join(source, "foreign"), "f")
	write(t, filepath.Join(source, "dangling"), "d")
	write(t, filepath.Join(dir, "elsewhere"), "e")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "elsewhere"), filepath.Join(target, "foreign")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(target, "dangling")); err != nil {
		t.Fatal(err)
	}

	rep, err := Apply(Tree{Source: source, Target: target}, resolveAll(ResFail), "")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(rep.Replaced) != 2 {
		t.Fatalf("Replaced = %v, want both stale links", rep.Replaced)
	}
	assertLink(t, filepath.Join(target, "foreign"), filepath.Join(source, "foreign"))
	assertLink(t, filepath.Join(target, "dangling"), filepath.Join(source, "dangling"))
	if got, err := os.ReadFile(filepath.Join(dir, "elsewhere")); err != nil || string(got) != "e" {
		t.Fatalf("foreign link target disturbed: %q, %v", got, err)
	}
}

func TestApplyConflicts(t *testing.T) {
	tests := []struct {
		name    string
		res     Resolution
		wantErr bool
		check   func(t *testing.T, source, target, backup string, rep Report)
	}{
		{name: "backup", res: ResBackup, check: func(t *testing.T, source, target, backup string, rep Report) {
			t.Helper()
			assertLink(t, filepath.Join(target, "file"), filepath.Join(source, "file"))
			mirror := filepath.Join(backup, filepath.Join(target, "file"))
			if got, err := os.ReadFile(mirror); err != nil || string(got) != "user" {
				t.Fatalf("backup mirror = %q, %v", got, err)
			}
			if len(rep.Backed) != 1 {
				t.Fatalf("Backed = %v", rep.Backed)
			}
		}},
		{name: "adopt", res: ResAdopt, check: func(t *testing.T, source, target, backup string, rep Report) {
			t.Helper()
			assertLink(t, filepath.Join(target, "file"), filepath.Join(source, "file"))
			if got, err := os.ReadFile(filepath.Join(source, "file")); err != nil || string(got) != "user" {
				t.Fatalf("adopted source = %q, %v (want the user's copy)", got, err)
			}
			if len(rep.Adopted) != 1 {
				t.Fatalf("Adopted = %v", rep.Adopted)
			}
		}},
		{name: "skip", res: ResSkip, check: func(t *testing.T, source, target, backup string, rep Report) {
			t.Helper()
			if got, err := os.ReadFile(filepath.Join(target, "file")); err != nil || string(got) != "user" {
				t.Fatalf("skipped file disturbed: %q, %v", got, err)
			}
			if len(rep.Skipped) != 1 {
				t.Fatalf("Skipped = %v", rep.Skipped)
			}
		}},
		{name: "fail", res: ResFail, wantErr: true, check: func(t *testing.T, source, target, backup string, rep Report) {
			t.Helper()
			if got, err := os.ReadFile(filepath.Join(target, "file")); err != nil || string(got) != "user" {
				t.Fatalf("file disturbed by failed apply: %q, %v", got, err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			source, target := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
			backup := filepath.Join(dir, "backup")
			write(t, filepath.Join(source, "file"), "template")
			write(t, filepath.Join(target, "file"), "user")

			rep, err := Apply(Tree{Source: source, Target: target}, resolveAll(tt.res), backup)
			if tt.wantErr != (err != nil) {
				t.Fatalf("Apply err = %v, wantErr %v", err, tt.wantErr)
			}
			tt.check(t, source, target, backup, rep)
		})
	}
}

func TestPlanReportsStates(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, filepath.Join(source, "fresh"), "")
	write(t, filepath.Join(source, "ok"), "")
	write(t, filepath.Join(source, "stale"), "")
	write(t, filepath.Join(source, "conflict"), "")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(source, "ok"), filepath.Join(target, "ok")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(target, "stale")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(target, "conflict"), "user")

	actions, err := Plan(Tree{Source: source, Target: target})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := map[string]State{"fresh": StateLink, "ok": StateOK, "stale": StateRelink, "conflict": StateConflict}
	if len(actions) != len(want) {
		t.Fatalf("got %d actions, want %d: %+v", len(actions), len(want), actions)
	}
	for _, a := range actions {
		if want[filepath.Base(a.Site)] != a.State {
			t.Errorf("%s: state = %v, want %v", a.Site, a.State, want[filepath.Base(a.Site)])
		}
	}
}

func TestApplyTargetPerm(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "src"), filepath.Join(dir, ".ssh")
	write(t, filepath.Join(source, "config"), "")

	if _, err := Apply(Tree{Source: source, Target: target, TargetPerm: 0o700}, resolveAll(ResFail), ""); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("target perm = %o, want 700", perm)
	}
}

func TestRestore(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	backup := filepath.Join(dir, "backup")
	write(t, filepath.Join(source, "file"), "template")
	write(t, filepath.Join(target, "file"), "user")

	if _, err := Apply(Tree{Source: source, Target: target}, resolveAll(ResBackup), backup); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	restored, err := Restore(backup)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	site := filepath.Join(target, "file")
	if len(restored) != 1 || restored[0] != site {
		t.Fatalf("restored = %v, want [%s]", restored, site)
	}
	if got, err := os.ReadFile(site); err != nil || string(got) != "user" {
		t.Fatalf("restored content = %q, %v", got, err)
	}
	if info, err := os.Lstat(site); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("site is still a symlink after restore: %v, %v", info, err)
	}
	// The backup set survives the restore for a repeat run.
	if _, err := os.Stat(filepath.Join(backup, site)); err != nil {
		t.Fatalf("backup consumed by restore: %v", err)
	}
}

func TestRestoreEmptySetErrors(t *testing.T) {
	if _, err := Restore(t.TempDir()); err == nil {
		t.Fatal("Restore of empty set succeeded, want error")
	}
}

func TestApplyMissingSourceErrors(t *testing.T) {
	dir := t.TempDir()
	_, err := Apply(Tree{Source: filepath.Join(dir, "absent"), Target: filepath.Join(dir, "dst")}, resolveAll(ResFail), "")
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want wrapped ErrNotExist", err)
	}
}

// TestApplyFileNeverAdoptsOverGenerated pins the guard that keeps a stale
// live file from being moved over a freshly generated destination: an adopt
// resolution against an ApplyFile conflict degrades to a backup, so the
// render survives and the stale copy stays recoverable.
func TestApplyFileNeverAdoptsOverGenerated(t *testing.T) {
	dir := t.TempDir()
	site := filepath.Join(dir, "live", "settings.json")
	dest := filepath.Join(dir, "profile", "settings.json")
	backup := filepath.Join(dir, "backup")
	write(t, site, "stale")
	write(t, dest, "fresh render")

	var rep Report
	if err := ApplyFile(site, dest, resolveAll(ResAdopt), backup, &rep); err != nil {
		t.Fatalf("ApplyFile: %v", err)
	}
	assertLink(t, site, dest)
	if got, err := os.ReadFile(dest); err != nil || string(got) != "fresh render" {
		t.Fatalf("generated destination = %q, %v (adopt reverted the render)", got, err)
	}
	if len(rep.Backed) != 1 || len(rep.Adopted) != 0 {
		t.Fatalf("Backed = %v, Adopted = %v; want the stale copy backed up", rep.Backed, rep.Adopted)
	}
	mirror := filepath.Join(backup, site)
	if got, err := os.ReadFile(mirror); err != nil || string(got) != "stale" {
		t.Fatalf("backup mirror = %q, %v", got, err)
	}
}

// TestApplyFileMarksConflictsGenerated pins that resolvers see ApplyFile
// conflicts flagged Generated, so interactive prompts can withhold adoption.
func TestApplyFileMarksConflictsGenerated(t *testing.T) {
	dir := t.TempDir()
	site := filepath.Join(dir, "live", "f")
	dest := filepath.Join(dir, "gen", "f")
	write(t, site, "stale")
	write(t, dest, "fresh")

	var saw bool
	resolve := func(c Conflict) (Resolution, error) {
		saw = c.Generated
		return ResSkip, nil
	}
	var rep Report
	if err := ApplyFile(site, dest, resolve, filepath.Join(dir, "b"), &rep); err != nil {
		t.Fatalf("ApplyFile: %v", err)
	}
	if !saw {
		t.Fatal("conflict was not marked Generated")
	}
}
