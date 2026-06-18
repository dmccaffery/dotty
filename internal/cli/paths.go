// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// dotty's on-disk layout is split for privacy: ConfigDir holds shareable
// configuration intended for a public dotfiles repository, DataDir holds
// PII-adjacent state (security-key aliases, SSH key stubs) intended for a
// separate private repository. XDG fallbacks are ~/.config and ~/.local/share
// on every OS, including darwin — deliberately not os.UserConfigDir, which
// would scatter dotfiles into ~/Library/Application Support.

// ConfigDir returns $XDG_CONFIG_HOME/dotty (or ~/.config/dotty), without
// creating it.
func ConfigDir() (string, error) {
	return xdgDir("XDG_CONFIG_HOME", ".config")
}

// DataDir returns $XDG_DATA_HOME/dotty (or ~/.local/share/dotty), without
// creating it. Anything written beneath it must use EnsureDir with 0o700 —
// the directory is a privacy boundary, not just a path.
func DataDir() (string, error) {
	return xdgDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

// xdgDir resolves an XDG base directory env var, ignoring non-absolute values
// as the XDG spec requires, and appends dotty's app directory.
func xdgDir(envVar, fallback string) (string, error) {
	base := os.Getenv(envVar)
	if !filepath.IsAbs(base) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, fallback)
	}
	return filepath.Join(base, "dotty"), nil
}

// EnsureDir creates path (and parents) if missing and enforces perm on the
// leaf even when it already existed with looser permissions.
func EnsureDir(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}

// AtomicWriteFile writes data to path via a temp file in the same directory
// and an os.Rename, so readers never observe a partial file. perm applies to
// the final file regardless of umask.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod %s: %w", tmp.Name(), err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", tmp.Name(), err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync %s: %w", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmp.Name(), path, err)
	}
	return nil
}
