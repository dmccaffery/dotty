// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package macos

import (
	"context"
	"path/filepath"
	"runtime"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// ApplySelections runs the chosen system tweaks best-effort: init has
// already succeeded by the time they run, and a failed Dock restart or
// declined sudo is not worth unwinding it over. A no-op off darwin.
func ApplySelections(ctx context.Context, ios cli.IOStreams, r Runner,
	groups []string, wallpaper string, piv bool, home string) {
	if runtime.GOOS != "darwin" {
		return
	}
	if len(groups) > 0 {
		if err := Apply(ctx, r, groups); err != nil {
			tui.Warnf(ios, "Could not apply macOS defaults: %v", err)
		} else {
			tui.Successf(ios, "Applied %d macOS defaults groups", len(groups))
		}
	}
	if wallpaper != "" {
		path := filepath.Join(home, ".local", "share", "wallpapers", wallpaper)
		if err := SetWallpaper(ctx, r, path); err != nil {
			tui.Warnf(ios, "Could not set the wallpaper: %v", err)
		} else {
			tui.Successf(ios, "Wallpaper set to %s", wallpaper)
		}
	}
	if piv {
		tui.Infof(ios, "Enforcing smart-card login needs sudo")
		if err := EnforcePIV(ctx, r); err != nil {
			tui.Warnf(ios, "Could not enforce smart-card login: %v", err)
		} else {
			tui.Successf(ios, "Smart-card (PIV) login required system-wide")
		}
	}
}
