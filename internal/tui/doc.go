// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

// Package tui provides dotty's interactive surface: a shared huh theme,
// prompt helpers (confirm, input, fuzzy select) that refuse to run without a
// terminal, styled notice printers, and the custom bubbletea models the
// security-key and signing-key commands use.
//
// All interaction renders on ErrOut so stdout stays clean for command output
// and pipes.
package tui
