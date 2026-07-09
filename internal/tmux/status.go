// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package tmux

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// Runner runs tmux. Output captures the child's stdout so nothing leaks onto
// a hook's captured stream.
type Runner interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Glyphs for the OSC (non-tmux) terminal-title fallback. Inside tmux the
// glyph and colour come from the tmux theme via the @agent_status token.
const (
	waitingGlyph   = "● "
	attentionGlyph = "󰂚 "
)

// StatusArgs builds the tmux argv recording state on the pane's window:
// waiting and attention set @agent_status, anything else unsets it.
func StatusArgs(pane, state string) []string {
	switch state {
	case "waiting", "attention":
		return []string{"set-window-option", "-t", pane, "@agent_status", state}
	default:
		return []string{"set-window-option", "-t", pane, "-u", "@agent_status"}
	}
}

// SetStatus applies state to the pane's window, discarding output and error.
func SetStatus(ctx context.Context, r Runner, pane, state string) {
	_, _ = r.Output(ctx, "tmux", StatusArgs(pane, state)...)
}

// Title returns the OSC 0 escape sequence titling the terminal with the
// basename of dir, prefixed by the state's glyph (bare basename otherwise).
func Title(state, dir string) string {
	title := filepath.Base(dir)
	switch state {
	case "waiting":
		title = waitingGlyph + title
	case "attention":
		title = attentionGlyph + title
	}
	return "\033]0;" + title + "\007"
}

// WriteTTY writes s to the controlling terminal, swallowing every error (the
// agent may have captured the hook's stdout, so the title must go to /dev/tty).
func WriteTTY(s string) {
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = io.WriteString(f, s)
}
