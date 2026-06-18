// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

// Package brewfile drives `brew bundle` and `brew trust` to keep a profile's
// Brewfile — and the machine — reproducible. All functions take the Brewfile
// path explicitly; resolving a profile to its Brewfile is the caller's job.
package brewfile
