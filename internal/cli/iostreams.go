// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package cli

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IOStreams bundles the three process streams so commands and prompts write to
// injectable destinations instead of the os globals. Command output (key
// material, proxied tool output) goes to Out; prompts and notices go to ErrOut
// so stdout stays clean for pipes.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// System returns the real process streams.
func System() IOStreams {
	return IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
}

// IsInteractive reports whether the streams can host an interactive prompt:
// In and ErrOut must both be terminals. Out is deliberately not consulted —
// prompts render on ErrOut, so `dotty ... | pbcopy` stays interactive.
func (s IOStreams) IsInteractive() bool {
	return IsTerminal(s.In) && IsTerminal(s.ErrOut)
}

// IsTerminal reports whether v is an *os.File attached to a terminal.
func IsTerminal(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
