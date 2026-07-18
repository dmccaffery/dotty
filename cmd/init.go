// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/fonts"
	"github.com/bitwise-media-group/dotty/internal/git"
	"github.com/bitwise-media-group/dotty/internal/linker"
	"github.com/bitwise-media-group/dotty/internal/macos"
	"github.com/bitwise-media-group/dotty/internal/profile"
	"github.com/bitwise-media-group/dotty/internal/scaffold"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
	"github.com/bitwise-media-group/dotty/internal/tui"
	"github.com/bitwise-media-group/dotty/internal/wizard"
)

var initFlags = wizard.Flags{}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new dotfiles repository and set up this machine.",
	Long: `Create a dotfiles repository from the template embedded in dotty, driven by
a short wizard: where repositories live, which optional tools and coding
agents to include, and how to seed the Brewfile. ghostty, oh-my-posh, vivid,
zsh, and git config are always included.

Nothing is written until a summary is confirmed. init then renders the
repository — including the profile (answers, Brewfile, and the per-profile
renders of anything machine-class-specific) under profiles/<name>, so
profiles travel with the repo and are shared across machines of the same
class — stages it with git (the first commit is left for you to sign), links
the home/ tree into your home directory, activates the profile (the
active-profile symlink is the only machine-local state), and installs the
lobe-icons glyph font. Files already in the way of a link are resolved per
--on-conflict; backups land under $XDG_DATA_HOME/dotty/backups and are
restorable with dotty dotfiles restore.

Re-running init against an existing repository and profile walks the same
interview again with the stored answers as the defaults, so a machine class
can be extended (or trimmed) later; keeping every answer re-renders and
re-links idempotently, and a repository in the legacy layout is migrated to
the current one. Without a terminal the stored answers are taken as-is, so
scripted re-runs never prompt. With --yes a re-run reuses every stored
answer and skips the confirmation summary, asking only which profile to use
plus any question the stored profile predates. A new machine adopts a fresh
clone the same way — run init from inside it, or point --repo at it.`,
	Example: `  dotty init
  dotty init --repo ~/Repos/dotfiles --addons=tmux,lsd --agents=claude-code --yes`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("on-conflict") {
			initFlags.OnConflict = ""
		}
		if initFlags.ProfileName == "" {
			initFlags.ProfileName = rootFlags.Profile
		}
		return runInit(cmd.Context(), cli.System(), initFlags)
	},
}

func init() {
	initCmd.Flags().StringVar(&initFlags.Repo, "repo", "",
		"dotfiles repository path (default <repos-dir>/dotfiles)")
	initCmd.Flags().StringVar(&initFlags.ReposDir, "repos-dir", "",
		"directory your repositories live in (default ~/Repos)")
	initCmd.Flags().StringVar(&initFlags.ProfileName, "profile-name", "",
		"dotty profile to create (default machine name)")
	initCmd.Flags().StringSliceVar(&initFlags.AddOns, "addons", nil,
		"optional add-ons: nvim,btop,k9s,lazygit,lsd,tmux,yazi")
	initCmd.Flags().StringSliceVar(&initFlags.Agents, "agents", nil,
		"coding agents: claude-code,codex,opencode,antigravity,grok")
	initCmd.Flags().BoolVar(&initFlags.DumpBrews, "dump-brews", false,
		"seed the Brewfile from the installed packages")
	initCmd.Flags().BoolVar(&initFlags.Marketplace, "marketplace", false,
		"add the bitwise skills marketplace to the selected agents")
	initCmd.Flags().BoolVar(&initFlags.Harden, "harden", false,
		"confine the coding agents: sandbox, credential-read denies, ask-first permissions")
	initCmd.Flags().BoolVar(&initFlags.SecurityKeys, "security-keys", false,
		"this machine class signs with hardware security keys")
	initCmd.Flags().StringVar(&initFlags.GitName, "git-name", "", "git identity name for the private git config")
	initCmd.Flags().StringVar(&initFlags.GitEmail, "git-email", "", "git identity email for the private git config")
	initCmd.Flags().StringSliceVar(&initFlags.AllowedSerials, "allowed-serials", nil,
		"restrict the profile to these security-key serials")
	initCmd.Flags().StringVar(&initFlags.Worktrees, "worktrees", "",
		"agent worktree location: a directory name inside each repo (default .worktrees) or an absolute path")
	initCmd.Flags().StringSliceVar(&initFlags.MacOSDefaults, "macos-defaults", nil,
		"macOS defaults groups to apply (see the wizard picklist; empty for none)")
	initCmd.Flags().StringVar(&initFlags.Wallpaper, "wallpaper", "", "wallpaper image from ~/.local/share/wallpapers")
	initCmd.Flags().BoolVar(&initFlags.PIV, "piv", false, "require smart-card (PIV) login system-wide")
	initCmd.Flags().BoolVar(&initFlags.SkipKeys, "skip-keys", false, "skip hardware key enrollment")
	_ = initCmd.Flags().MarkHidden("skip-keys")
	initCmd.Flags().StringVar(&initFlags.OnConflict, "on-conflict", "backup",
		"existing-file resolution: backup, adopt, skip, or fail")
	initCmd.Flags().BoolVar(&initFlags.Yes, "yes", false,
		"skip the confirmation summary and reuse stored answers; only unanswered questions are asked")
	initCmd.Flags().BoolVar(&initFlags.SkipFont, "skip-font", false, "skip the lobe-icons font download")
	initCmd.Flags().BoolVar(&initFlags.SkipGit, "skip-git", false, "skip git init")
	_ = initCmd.Flags().MarkHidden("skip-font")
	_ = initCmd.Flags().MarkHidden("skip-git")
	rootCmd.AddCommand(initCmd)
}

// runInit is the whole init flow: collect answers, confirm, then render,
// git, link, activate, identity, keys, font, macOS — in an order where every
// question that gates a write precedes the first write.
func runInit(ctx context.Context, ios cli.IOStreams, flags wizard.Flags) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	answers, repo, rerun, err := wizard.Collect(ios, flags, home)
	if errors.Is(err, tui.ErrAborted) {
		return nil // esc backs out before anything is written
	}
	if err != nil {
		return err
	}

	if !flags.Yes {
		ok, err := wizard.ConfirmSummary(ios, answers, repo, rerun)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	runner := newRunner(ios)
	pruned, err := scaffold.RenderRepository(ctx, ios, runner, answers, repo, home)
	if err != nil {
		return err
	}

	if !flags.SkipGit {
		if err := git.InitRepo(ctx, runner, repo); err != nil {
			return err
		}
	}

	// Link before activating: the live profile path is a symlink into the
	// repository, and Activate needs to see it.
	report, backupDir, err := linker.LinkHome(ios, answers, repo, home, flags.OnConflict)
	if err != nil {
		return err
	}
	linker.Summarize(ios, report, backupDir)
	linker.PruneSites(ios, home, pruned)

	configDir, err := cli.ConfigDir()
	if err != nil {
		return err
	}
	if _, err := profile.Activate(ctx, runner, configDir, answers.ProfileName); err != nil {
		return err
	}
	tui.Successf(ios, "Profile %s active", answers.ProfileName)
	tui.Infof(ios, "Install everything with: dotty brewfile sync")

	if err := git.EnsureIdentityFile(ios, flags.GitName, flags.GitEmail, answers.SecurityKeys, home); err != nil {
		return err
	}
	if answers.SecurityKeys {
		if err := setUpSecurityKeys(ctx, ios, flags, home); err != nil {
			return err
		}
		if err := setUpKeyAllowlist(ctx, ios, flags, &answers, repo); err != nil {
			return err
		}
	}

	if !flags.SkipFont {
		fonts.Install(ctx, ios, home)
	}

	macos.ApplySelections(ctx, ios, runner, answers.MacOSDefaults, answers.Wallpaper, answers.PIV, home)

	tui.Infof(ios, "Next: cd %s && git commit -m %q, then restart your terminal.", repo, "chore: initial dotfiles")
	return nil
}

// setUpSecurityKeys wires the hardware-key plumbing: the dotty-ssh-askpass
// applet symlink OpenSSH PIN prompts route through (its basename is how
// dispatchArgs recognizes the invocation), and optionally a signing key —
// imported from existing stubs or enrolled fresh — via the same flows the
// signing-key verbs run. It stays in cmd because those flows (enroll,
// import, trust) live with the signing-key verbs.
func setUpSecurityKeys(ctx context.Context, ios cli.IOStreams, flags wizard.Flags, home string) error {
	dataDir, err := cli.DataDir()
	if err != nil {
		return err
	}
	if err := linkAskpassApplet(ios, dataDir); err != nil {
		return err
	}

	if flags.SkipKeys {
		return nil
	}

	// Keys enrolled earlier — stubs in the private data dir, from a previous
	// init, the legacy tooling, or a synced private repo — beat re-importing.
	// The profile's allowlist still filters what this machine class may use.
	existing, err := existingKeyRefs(dataDir)
	if err != nil {
		return err
	}
	if !ios.IsInteractive() {
		if len(existing) > 0 {
			return useExistingKeys(ctx, ios, home, existing)
		}
		return nil
	}
	// --yes reuses the machine's enrollment the way a scripted re-run does;
	// with nothing on disk the question has no stored answer, so it is asked.
	if flags.Yes && len(existing) > 0 {
		return useExistingKeys(ctx, ios, home, existing)
	}

	options := make([]tui.Option, 0, 4)
	if len(existing) > 0 {
		options = append(options, tui.Option{
			Label: fmt.Sprintf("use the profile's existing keys (%d on disk)", len(existing)),
			Value: "existing",
		})
	}
	options = append(options,
		tui.Option{Label: "import an existing resident-key stub", Value: "import"},
		tui.Option{Label: "create a new key on a YubiKey", Value: "new"},
		tui.Option{Label: "later (dotty signing-key new / import)", Value: "later"},
	)
	picked, err := tui.FuzzySelect(ios, "Set up a signing key on this machine?", options)
	if errors.Is(err, tui.ErrAborted) || errors.Is(err, tui.ErrNotInteractive) {
		return nil
	}
	if err != nil {
		return err
	}
	switch picked {
	case "existing":
		return useExistingKeys(ctx, ios, home, existing)
	case "import":
		path, err := tui.Input(ios, "Where are the key stubs?", "~/keys or a stub file", nil)
		if err != nil || path == "" {
			return nil
		}
		if path, err = cli.ExpandHome(path); err != nil {
			return err
		}
		return importSigningKeys(ctx, ios, path, "", false)
	case "new":
		return enrollSigningKey(ctx, ios, "ed25519", "", "")
	}
	return nil
}

// linkAskpassApplet points the dotty-ssh-askpass applet symlink at the
// running binary; the basename is how dispatchArgs recognizes OpenSSH PIN
// prompts and routes them to pinentry-mac for keychain caching.
func linkAskpassApplet(ios cli.IOStreams, dataDir string) error {
	if err := cli.EnsureDir(dataDir, 0o700); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate dotty binary: %w", err)
	}
	askpass := filepath.Join(dataDir, "dotty-ssh-askpass")
	if existing, err := os.Readlink(askpass); err == nil && existing == self {
		return nil
	}
	if err := os.Remove(askpass); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("replace %s: %w", askpass, err)
	}
	if err := os.Symlink(self, askpass); err != nil {
		return fmt.Errorf("link %s: %w", askpass, err)
	}
	tui.Successf(ios, "Linked %s so pinentry-mac caches YubiKey PINs", askpass)
	return nil
}

// existingKeyRefs returns every signing-key stub on disk, filtered by the
// active profile's allowlist — the keys this machine can use without
// importing or enrolling anything.
func existingKeyRefs(dataDir string) ([]signingkey.KeyRef, error) {
	refs, err := signingkey.Scan(dataDir, nil, "")
	if err != nil {
		return nil, err
	}
	return filterAllowedRefs(refs)
}

// useExistingKeys wires the already-enrolled keys in: trust them so git can
// verify what they sign (best-effort — identity may still be missing), and
// link the stable ssh identity path at the preferred stub.
func useExistingKeys(ctx context.Context, ios cli.IOStreams, home string, refs []signingkey.KeyRef) error {
	runner := newRunner(ios)
	if err := trustKeys(ctx, ios, runner, "", refs); err != nil {
		tui.Warnf(ios, "could not update allowed_signers: %v", err)
	}

	preferDefaultKeyType(refs)
	linkPath := signingkey.DefaultLinkPath(home)
	if err := signingkey.Link(linkPath, refs[0]); err != nil {
		return err
	}
	tui.Successf(ios, "Using %d existing signing key(s); %s -> YubiKey %s", len(refs), linkPath, refs[0].Serial)
	return nil
}

// setUpKeyAllowlist restricts the profile to specific security-key serials —
// after enrollment on purpose, so a freshly enrolled key is connected and
// offered. The list is part of the profile's answers: it travels with the
// repository and activating another profile swaps it.
func setUpKeyAllowlist(ctx context.Context, ios cli.IOStreams, flags wizard.Flags,
	answers *scaffold.Answers, repo string) error {
	// The flag path was already validated and persisted with the render, and
	// a non-nil list — including the answered-empty one a declined offer
	// stores — means the question was answered before; only a profile that
	// has never been asked gets the interactive offer.
	if answers.AllowedSerials != nil || !ios.IsInteractive() || flags.SkipKeys {
		return nil
	}
	prompt := fmt.Sprintf("Restrict profile %s to specific security keys?", answers.ProfileName)
	ok, err := tui.Confirm(ios, prompt,
		"other keys are refused for signing, linking, enrollment, and import on this machine class")
	if err != nil {
		return err
	}
	repoProfileDir := profile.Dir(scaffold.ProfilesDir(repo), answers.ProfileName)
	if !ok {
		// Remember the "no" as an answered-empty allowlist (empty means
		// unrestricted) so re-runs stop asking; dotty security-key allow
		// can still restrict the profile later.
		answers.AllowedSerials = []string{}
		return scaffold.SaveAnswers(repoProfileDir, *answers)
	}
	store, err := keyStore()
	if err != nil {
		return err
	}
	serials, err := pickAllowSerials(ctx, ios, store, *answers)
	if err != nil || len(serials) == 0 {
		return err
	}
	slices.Sort(serials)
	answers.AllowedSerials = serials

	if err := scaffold.SaveAnswers(repoProfileDir, *answers); err != nil {
		return err
	}
	tui.Successf(ios, "Profile %s allows only these security keys: %s",
		answers.ProfileName, strings.Join(answers.AllowedSerials, ", "))
	return nil
}
