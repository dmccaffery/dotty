// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/env"
)

// EnvFlags holds the flags shared by the env verbs.
type EnvFlags struct {
	// Namespace is persistent on the noun so it parses in both positions:
	// `dotty env --namespace=aws add KEY` and `dotty env add --namespace=aws KEY`.
	Namespace string
}

var envFlags = EnvFlags{}

var envCmd = &cobra.Command{
	Use:   "env <verb>",
	Short: "Store and inject credentials from the macOS Keychain.",
	Long: `Manage generic credentials in the macOS login keychain and inject them into
templates and processes — the way the 1Password CLI does, but with no external
service. Secrets are grouped into namespaces; each namespace is a single
keychain item under the service name "dotty:<namespace>". get reads one value
(like op read), use fills a template (like op inject), and run launches a
process with the namespace's secrets in its environment (like op run).`,
	Example: `  dotty env add --namespace=aws AWS_ACCESS_KEY_ID
  dotty env list --namespace=aws
  dotty env get --namespace=aws AWS_ACCESS_KEY_ID
  dotty env run --namespace=aws -- aws s3 ls`,
}

// newEnvStore builds the credential store backed by the platform keychain.
func newEnvStore(ios cli.IOStreams) *env.Store {
	return env.NewStore(env.NewKeychain(newRunner(ios)))
}

// defaultEnvFile is the project-local template an env verb falls back to when
// invoked with neither a --namespace nor an --in-file: the .env.dotty in the
// working directory. It lets `dotty env use` and `dotty env run` work with no
// arguments inside a project that ships one.
const defaultEnvFile = ".env.dotty"

// errNoDefaultEnvFile is returned by defaultEnvFileOrErr when the working
// directory has no defaultEnvFile to fall back to. Callers pair it with the
// command's usage so a bare invocation explains itself.
var errNoDefaultEnvFile = fmt.Errorf(
	"no %s in the current directory; pass --in-file=<file> or --namespace=<ns>", defaultEnvFile)

// defaultEnvFileOrErr resolves the implicit template for an env verb run with
// neither a --namespace nor an --in-file. It returns defaultEnvFile when it
// exists in the working directory, errNoDefaultEnvFile when it does not, or a
// stat error for anything else.
func defaultEnvFileOrErr() (string, error) {
	switch _, err := os.Stat(defaultEnvFile); {
	case err == nil:
		return defaultEnvFile, nil
	case errors.Is(err, os.ErrNotExist):
		return "", errNoDefaultEnvFile
	default:
		return "", fmt.Errorf("stat %s: %w", defaultEnvFile, err)
	}
}

func init() {
	envCmd.PersistentFlags().StringVar(&envFlags.Namespace, "namespace", "default",
		"credential namespace to operate on")
	rootCmd.AddCommand(envCmd)
}
