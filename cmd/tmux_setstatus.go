// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/tmux"
)

// tmuxSetStatusCmd is the coding-agent status indicator agent lifecycle hooks
// invoke (Claude Code / Codex: Stop→waiting, Notification/PermissionRequest→
// attention, everything else→clear), never a user at the prompt, hence
// hidden. Hooks treat a non-zero exit as a failure, so the command is
// no-op-safe: every branch swallows errors and RunE always returns nil.
var tmuxSetStatusCmd = &cobra.Command{
	Use:           "set-status [waiting|attention|clear]",
	Short:         "Set the coding-agent status indicator (internal).",
	Hidden:        true,
	Args:          cobra.ArbitraryArgs, // never fail on argv shape
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		state := "clear"
		if len(args) > 0 {
			state = args[0]
		}
		if os.Getenv("TMUX") != "" {
			tmux.SetStatus(cmd.Context(), newRunner(cli.System()), os.Getenv("TMUX_PANE"), state)
			return nil
		}
		dir := os.Getenv("PWD")
		if dir == "" {
			dir, _ = os.Getwd()
		}
		tmux.WriteTTY(tmux.Title(state, dir))
		return nil
	},
}

func init() {
	tmuxCmd.AddCommand(tmuxSetStatusCmd)
}
