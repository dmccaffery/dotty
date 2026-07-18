// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

// Package linker symlinks a dotfiles repository's trees into their live
// targets: whole files and directories are linked
// folded, existing real directories are descended into and their children
// linked individually, and stale symlinks are replaced in place. Real files
// found in the way are conflicts the caller resolves — backed up under a
// mirror of their absolute path so they stay recoverable, adopted into the
// repository, or skipped.
package linker
