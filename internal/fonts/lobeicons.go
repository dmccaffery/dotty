// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package fonts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bitwise-media-group/dotty/internal/cli"
)

// lobe-icons maps 300+ AI/LLM brand logos onto U+F4000–U+F47FF, a private-use
// range no Nerd Font occupies. The tmux agent-window glyphs and the ghostty
// font-codepoint-map in the dotfiles template both assume this exact release,
// so URL, checksum, and the codepoints baked into internal/tmux are a matched
// set — re-verify codepoints.json when bumping the version.
const (
	// LobeIconsFile is the installed filename, and the family name terminals
	// map the codepoint range to.
	LobeIconsFile = "lobe-icons.ttf"
	// LobeIconsURL is the pinned upstream release asset.
	LobeIconsURL = "https://github.com/hschne/lobe-icons-font/releases/download/v5.13.0/lobe-icons.ttf"
)

// pinSHA256 is the checksum of the pinned release asset; a var only so tests
// can serve a stand-in body.
var pinSHA256 = "bc253a673d6c7beaf800fc872d4473b532c2474aba33b895b3e721ae554b0c91"

// Doer issues one HTTP request; *http.Client satisfies it.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// InstallLobeIcons downloads the pinned lobe-icons release into dir unless
// already present, verifying the checksum before anything lands on disk. It
// reports whether a download happened; (false, nil) means the font was
// already installed.
func InstallLobeIcons(ctx context.Context, client Doer, url, dir string) (bool, error) {
	path := filepath.Join(dir, LobeIconsFile)
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("inspect %s: %w", path, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("request %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("download %s: %w", url, err)
	}

	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != pinSHA256 {
		return false, fmt.Errorf("verify %s: checksum %s does not match the pinned release", url, got)
	}
	if err := cli.EnsureDir(dir, 0o755); err != nil {
		return false, err
	}
	if err := cli.AtomicWriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}
