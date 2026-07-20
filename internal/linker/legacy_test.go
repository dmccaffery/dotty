// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package linker

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRetireLegacy(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	backup := filepath.Join(dir, "backup")

	write(t, filepath.Join(home, ".gitconfig"), "[user]\n\tname = old\n")
	if err := os.Symlink("/nowhere", filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".config", "git", "config"), "rendered")

	var rep Report
	if err := retireLegacy(home, backup, &rep); err != nil {
		t.Fatalf("retireLegacy() error = %v", err)
	}

	want := []string{filepath.Join(home, ".gitconfig"), filepath.Join(home, ".zshrc")}
	if !slices.Equal(rep.Retired, want) {
		t.Errorf("Retired = %v, want %v", rep.Retired, want)
	}
	for _, site := range want {
		if _, err := os.Lstat(site); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Lstat(%s) error = %v, want ErrNotExist", site, err)
		}
	}

	// The backup mirrors each site's absolute path, so Restore can put it back.
	mirrored := filepath.Join(backup, filepath.Join(home, ".gitconfig"))
	got, err := os.ReadFile(mirrored)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", mirrored, err)
	}
	if string(got) != "[user]\n\tname = old\n" {
		t.Errorf("backup content = %q, want the original ~/.gitconfig", got)
	}
	if _, err := os.Lstat(filepath.Join(backup, filepath.Join(home, ".zshrc"))); err != nil {
		t.Errorf("backed-up symlink missing: %v", err)
	}

	// Untouched: files that are not legacy shadows.
	if _, err := os.Stat(filepath.Join(home, ".config", "git", "config")); err != nil {
		t.Errorf("non-legacy file was touched: %v", err)
	}
}

func TestRetireLegacyIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	write(t, filepath.Join(home, ".zshrc"), "old")

	var first Report
	if err := retireLegacy(home, filepath.Join(dir, "b1"), &first); err != nil {
		t.Fatalf("retireLegacy() error = %v", err)
	}
	var second Report
	if err := retireLegacy(home, filepath.Join(dir, "b2"), &second); err != nil {
		t.Fatalf("retireLegacy() second run error = %v", err)
	}
	if len(second.Retired) != 0 {
		t.Errorf("second run Retired = %v, want none", second.Retired)
	}
}
