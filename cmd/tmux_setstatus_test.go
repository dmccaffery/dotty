// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import "testing"

// TestTmuxSetStatusNeverFails pins the hook contract: whatever the argv shape
// or environment, the command exits 0. The tmux cases point $TMUX at a socket
// that cannot exist so the real tmux client fails (or is absent entirely) and
// the error is swallowed; the non-tmux cases fall back to the terminal title,
// whose /dev/tty write fails harmlessly when the test has no terminal.
func TestTmuxSetStatusNeverFails(t *testing.T) {
	tests := []struct {
		name string
		tmux string // $TMUX value; empty selects the terminal-title branch
		args []string
	}{
		{name: "no state, no tmux", args: []string{"tmux", "set-status"}},
		{name: "unknown state, no tmux", args: []string{"tmux", "set-status", "bogus"}},
		{
			name: "waiting, unreachable tmux",
			tmux: "/dev/null/no-such-socket,0,0",
			args: []string{"tmux", "set-status", "waiting"},
		},
		{
			name: "extra args, unreachable tmux",
			tmux: "/dev/null/no-such-socket,0,0",
			args: []string{"tmux", "set-status", "attention", "extra"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMUX", tt.tmux)
			t.Setenv("TMUX_PANE", "%999")

			if err := execDotty(t, tt.args...); err != nil {
				t.Errorf("execute %v = %v, want nil", tt.args, err)
			}
		})
	}
}
