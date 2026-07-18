// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package macos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Runner runs the system tools the settings go through. RunInteractive
// covers sudo, which needs the terminal for its password prompt.
type Runner interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	RunInteractive(ctx context.Context, name string, args ...string) error
}

// Group is one selectable set of defaults; Restart names the services that
// must relaunch to pick the writes up.
type Group struct {
	ID      string
	Label   string
	writes  [][]string
	Restart []string
}

// Groups are the curated defaults, in menu order. IDs are what Answers
// persists, so renaming one orphans saved selections.
var Groups = []Group{
	{
		ID: "keyboard", Label: "keyboard: key repeat instead of the accent popup",
		writes: [][]string{{"-g", "ApplePressAndHoldEnabled", "-bool", "false"}},
	},
	{
		ID: "menu-bar", Label: "menu bar: always visible, except in full screen",
		writes: [][]string{
			{"-g", "_HIHideMenuBar", "-bool", "false"},
			{"-g", "AppleMenuBarVisibleInFullscreen", "-bool", "false"},
			{"com.apple.controlcenter", "AutoHideMenuBarOption", "-int", "2"},
		},
		Restart: []string{"SystemUIServer"},
	},
	{
		ID: "trackpad", Label: "trackpad: three-finger drag + drag windows on gesture",
		writes: [][]string{
			{"com.apple.AppleMultitouchTrackpad", "TrackpadThreeFingerDrag", "-bool", "true"},
			{"-g", "NSWindowShouldDragOnGesture", "-bool", "true"},
		},
	},
	{
		ID: "finder", Label: "finder: clean desktop, path/status bars, no extension warnings",
		writes: [][]string{
			{"com.apple.TimeMachine", "DoNotOfferNewDisksForBackup", "-bool", "true"},
			{"com.apple.finder", "ShowExternalHardDrivesOnDesktop", "-bool", "false"},
			{"com.apple.finder", "ShowHardDrivesOnDesktop", "-bool", "false"},
			{"com.apple.finder", "ShowRemovableMediaOnDesktop", "-bool", "false"},
			{"com.apple.finder", "ShowMountedServersOnDesktop", "-bool", "false"},
			{"com.apple.finder", "CreateDesktop", "-bool", "false"},
			{"com.apple.desktopservices", "DSDontWriteNetworkStores", "-bool", "true"},
			{"com.apple.finder", "ShowPathbar", "-bool", "true"},
			{"com.apple.finder", "ShowStatusBar", "-bool", "true"},
			{"com.apple.finder", "FXEnableExtensionChangeWarning", "-bool", "false"},
		},
		Restart: []string{"Finder"},
	},
	{
		ID: "screenshots", Label: "screenshots: save as png",
		writes: [][]string{{"com.apple.screencapture", "type", "-string", "png"}},
	},
	{
		ID: "software-update", Label: "software update: check daily",
		writes: [][]string{{"com.apple.SoftwareUpdate", "ScheduleFrequency", "-int", "1"}},
	},
	{
		ID: "spaces", Label: "spaces: per-display, never rearranged",
		writes: [][]string{
			{"com.apple.spaces", "spans-displays", "-bool", "false"},
			{"com.apple.dock", "mru-spaces", "-bool", "false"},
		},
		Restart: []string{"Dock"},
	},
	{
		ID: "dock", Label: "dock: sizing and minimize-to-application",
		writes: [][]string{
			{"com.apple.dock", "autohide", "-bool", "false"},
			{"com.apple.dock", "largesize", "-float", "96"},
			{"com.apple.dock", "minimize-to-application", "-bool", "true"},
			{"com.apple.dock", "tilesize", "-float", "48"},
		},
		Restart: []string{"Dock"},
	},
	{
		ID: "animations", Label: "animations: disable window animations",
		writes: [][]string{{"-g", "NSAutomaticWindowAnimationsEnabled", "-bool", "false"}},
	},
	{
		ID: "gpg-keychain", Label: "gpg: cache PINs in the macOS keychain",
		writes: [][]string{
			{"org.gpgtools.common", "UseKeychain", "-bool", "yes"},
			{"org.gpgtools.common", "DisableKeychain", "-bool", "no"},
		},
	},
}

// Apply writes the selected groups' defaults and restarts the affected
// services once, after all writes. Unknown ids are an error, not a silent
// skip.
func Apply(ctx context.Context, r Runner, ids []string) error {
	var restarts []string
	for _, id := range ids {
		i := slices.IndexFunc(Groups, func(g Group) bool { return g.ID == id })
		if i < 0 {
			return fmt.Errorf("unknown defaults group %q", id)
		}
		for _, w := range Groups[i].writes {
			if _, err := r.Output(ctx, "defaults", append([]string{"write"}, w...)...); err != nil {
				return fmt.Errorf("defaults write %s: %w", strings.Join(w, " "), err)
			}
		}
		restarts = append(restarts, Groups[i].Restart...)
	}
	slices.Sort(restarts)
	for _, service := range slices.Compact(restarts) {
		_, _ = r.Output(ctx, "killall", service) // not running is fine
	}
	return nil
}

// EnforcePIV requires smart-card login system-wide. sudo prompts on the
// terminal; the caller confirms first — this can lock a machine without an
// enrolled card out.
func EnforcePIV(ctx context.Context, r Runner) error {
	for _, args := range [][]string{
		{"defaults", "write", "/Library/Preferences/com.apple.security.smartcard", "enforceSmartCard", "-bool", "true"},
		{"defaults", "write", "/Library/Preferences/com.apple.security.smartcard", "allowUnmappedUsers", "-int", "1"},
	} {
		if err := r.RunInteractive(ctx, "sudo", args...); err != nil {
			return fmt.Errorf("sudo %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

// SetWallpaper points every desktop at the image.
func SetWallpaper(ctx context.Context, r Runner, path string) error {
	script := fmt.Sprintf(`tell application "System Events" to tell every desktop to set picture to %q`, path)
	if _, err := r.Output(ctx, "osascript", "-e", script); err != nil {
		return fmt.Errorf("set wallpaper %s: %w", path, err)
	}
	return nil
}

// Wallpapers lists the images under dir (the conventional
// ~/.local/share/wallpapers, populated from the user's private repo — dotty
// distributes none). A missing directory is an empty list, not an error.
func Wallpapers(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var images []string
	for _, e := range entries {
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".png", ".jpg", ".jpeg", ".heic", ".tiff":
			if !e.IsDir() {
				images = append(images, e.Name())
			}
		}
	}
	return images
}
