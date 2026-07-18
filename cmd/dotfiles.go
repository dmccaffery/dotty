// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/profile"
	"github.com/bitwise-media-group/dotty/internal/scaffold"
)

// DotfilesFlags holds the flags shared by the dotfiles verbs.
type DotfilesFlags struct {
	Repo string
}

var dotfilesFlags = DotfilesFlags{}

var dotfilesCmd = &cobra.Command{
	Use:   "dotfiles",
	Short: "Operate on the dotfiles repository dotty init generated.",
	Long: `Link, inspect, and recover the dotfiles repository created by dotty init —
or any repository with the same layout: a home/ tree of $HOME-relative
entries plus profiles/ directories whose profile.json records the choices
that built them.`,
	Example: `  dotty dotfiles status
  dotty dotfiles link --on-conflict=backup
  dotty dotfiles restore`,
}

func init() {
	dotfilesCmd.PersistentFlags().StringVar(&dotfilesFlags.Repo, "repo", "",
		"dotfiles repository (default: found via $REPOS_DIR)")
	rootCmd.AddCommand(dotfilesCmd)
}

// resolveDotfilesRepo locates the repository the verbs operate on — --repo,
// the enclosing dotfiles repository, or the paths the active profile's
// answers stored — and loads those answers, which own everything that
// varies across machine classes.
func resolveDotfilesRepo() (string, scaffold.Answers, error) {
	configDir, err := cli.ConfigDir()
	if err != nil {
		return "", scaffold.Answers{}, err
	}
	activeDir, err := profile.ActiveDir(configDir)
	if err != nil {
		return "", scaffold.Answers{}, fmt.Errorf("no active profile; run dotty init first: %w", err)
	}
	answers, err := scaffold.LoadAnswers(activeDir)
	if err != nil {
		return "", scaffold.Answers{}, fmt.Errorf(
			"active profile has no %s; run dotty init first: %w", scaffold.AnswersFile, err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", scaffold.Answers{}, fmt.Errorf("resolve home: %w", err)
	}
	repo := dotfilesFlags.Repo
	if repo == "" {
		repo = scaffold.EnclosingRepo()
	}
	if repo == "" {
		reposDir := scaffold.ExpandTilde(answers.ReposDir, home)
		if repo = scaffold.ExpandTilde(answers.Repo, home); repo == "" {
			repo = filepath.Join(reposDir, "dotfiles")
		} else if !filepath.IsAbs(repo) {
			repo = filepath.Join(reposDir, repo)
		}
	}
	if repo, err = cli.ExpandHome(repo); err != nil {
		return "", scaffold.Answers{}, err
	}
	if _, err := os.Stat(scaffold.HomeDir(repo)); err != nil {
		if _, legacyErr := os.Stat(filepath.Join(repo, "stow")); legacyErr == nil {
			return "", scaffold.Answers{},
				fmt.Errorf("%s uses the legacy layout; run dotty init to migrate it first", repo)
		}
		return "", scaffold.Answers{},
			fmt.Errorf("%s is not a dotfiles repository (no home tree); pass --repo: %w", repo, err)
	}
	return repo, answers, nil
}
