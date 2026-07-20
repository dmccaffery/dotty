// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package linker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// legacyShadows names the files older setups leave directly in $HOME that
// override or bypass the always-rendered configuration without ever occupying
// a link site, so conflict resolution never sees them: git reads ~/.gitconfig
// after ~/.config/git/config, so its stale values silently win, and the bare
// zsh startup files predate the ~/.zshenv ZDOTDIR redirect into ~/.config/zsh.
// ~/.zshenv itself is a link site, so the ordinary conflict flow covers it.
var legacyShadows = []string{".gitconfig", ".zshrc", ".zprofile", ".zlogin", ".zlogout"}

// retireLegacy moves each legacy shadow out of $HOME into the backup set,
// recording the moves in rep.Retired. Always a backup, never a plain removal —
// the file predates dotty, so its content (identities, tokens, PATH tweaks)
// may exist nowhere else, and the mirror keeps it restorable with
// dotty dotfiles restore.
func retireLegacy(home, backupRoot string, rep *Report) error {
	for _, name := range legacyShadows {
		site := filepath.Join(home, name)
		if _, err := os.Lstat(site); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("inspect %s: %w", site, err)
		}
		dst := filepath.Join(backupRoot, strings.TrimPrefix(site, string(filepath.Separator)))
		if err := move(site, dst); err != nil {
			return fmt.Errorf("retire %s: %w", site, err)
		}
		rep.Retired = append(rep.Retired, site)
	}
	return nil
}
