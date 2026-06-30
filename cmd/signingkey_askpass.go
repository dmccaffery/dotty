// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
)

// signingKeyAskPassCmd is the $SSH_ASKPASS bridge `signing-key sign` points
// ssh-keygen at. It is invoked as `dotty <prompt>` with DOTTY_ASKPASS=1 set, so
// the argv dispatcher routes it here; it is never run by hand, hence hidden.
var signingKeyAskPassCmd = &cobra.Command{
	Use:    "ask-pass [prompt]",
	Short:  "Bridge OpenSSH PIN prompts to pinentry-mac (internal).",
	Hidden: true,
	// ssh passes a single prompt that may begin with '-'; take it verbatim
	// rather than parsing it as flags.
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		var prompt string
		if len(args) > 0 {
			prompt = args[0]
		}
		reply := signingkey.AskPassReply(cmd.Context(), newRunner(ios), prompt, os.Getenv(signingkey.KeyInfoEnv))
		_, _ = fmt.Fprintln(ios.Out, reply)
		return nil
	},
}

func init() {
	signingKeyCmd.AddCommand(signingKeyAskPassCmd)
}
