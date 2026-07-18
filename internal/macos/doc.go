// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

// Package macos applies macOS system preferences: curated groups of
// `defaults write` settings, the desktop wallpaper, and PIV smart-card
// enforcement. Everything shells out through a Runner, so the package builds
// and tests everywhere — callers gate on runtime.GOOS before invoking it on
// a machine.
package macos
