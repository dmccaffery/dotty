// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package profile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bitwise-media-group/dotty/internal/brewfile"
)

// Activate points the active-profile symlink at the named profile and returns
// the profile's directory. The swap is atomic — a temp symlink renamed over
// the old one — so the link never dangles mid-switch. The symlink target is
// the bare profile name (relative), which survives a home-directory move and
// reads cleanly in a dotfiles repository.
//
// A profile activated without a Brewfile gets one dumped from the currently
// installed brews, so `dotty brewfile ...` works immediately after.
func Activate(ctx context.Context, r brewfile.Runner, configDir, name string) (string, error) {
	if !Exists(configDir, name) {
		return "", fmt.Errorf("profile %q: %w", name, ErrNotFound)
	}
	tmp := filepath.Join(configDir, ".active-profile.tmp")
	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("clear stale temp symlink: %w", err)
	}
	if err := os.Symlink(name, tmp); err != nil {
		return "", fmt.Errorf("create temp symlink: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(configDir, activeLink)); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("swap active-profile symlink: %w", err)
	}

	dir := Dir(configDir, name)
	if _, err := os.Stat(BrewfilePath(dir)); errors.Is(err, fs.ErrNotExist) {
		if err := brewfile.Dump(ctx, r, BrewfilePath(dir), false, false); err != nil {
			return dir, fmt.Errorf("dump Brewfile for fresh profile: %w", err)
		}
	}
	return dir, nil
}
