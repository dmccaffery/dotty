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

// signingKeyAskPassCmd is the $SSH_ASKPASS bridge that turns an OpenSSH FIDO PIN
// prompt into a pinentry-mac dialog. dotty's own sign path points ssh-keygen at
// it with the DOTTY_ASKPASS=1 sentinel; a globally-exported
// SSH_ASKPASS=<dir>/dotty-ssh-askpass routes every other prompt here by
// argv[0] basename — including yes/no confirmations such as the
// host-authenticity check, which become pinentry CONFIRM dialogs rather than
// PIN entries. The argv dispatcher rewrites the call either way, so it is
// never run by hand, hence hidden.
var signingKeyAskPassCmd = &cobra.Command{
	Use:    "ask-pass [prompt]",
	Short:  "Bridge OpenSSH askpass prompts to pinentry-mac (internal).",
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
		reply := signingkey.AskPassReply(cmd.Context(), newRunner(ios), prompt,
			os.Getenv(signingkey.PromptHintEnv), os.Getenv(signingkey.KeyInfoEnv), os.ReadFile)
		_, _ = fmt.Fprintln(ios.Out, reply)
		return nil
	},
}

func init() {
	signingKeyCmd.AddCommand(signingKeyAskPassCmd)
}
