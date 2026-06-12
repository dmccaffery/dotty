// MIT License
//
// Copyright (c) 2026 Bitwise Media Group
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package brewfile

import (
	"context"
	"fmt"
	"strings"
)

// Runner executes brew on behalf of this package; tests substitute a fake.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) error
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Kind is a brew bundle entry type. Its string form doubles as the brew CLI
// flag name.
type Kind string

// The entry types `brew bundle add` accepts.
const (
	KindFormula Kind = "formula"
	KindCask    Kind = "cask"
	KindTap     Kind = "tap"
	KindVSCode  Kind = "vscode"
	KindGo      Kind = "go"
	KindCargo   Kind = "cargo"
	KindUV      Kind = "uv"
	KindFlatpak Kind = "flatpak"
	KindKrew    Kind = "krew"
	KindNPM     Kind = "npm"
)

// Trustable reports whether `brew trust` applies to this kind — Homebrew
// trusts only taps, formulae, and casks.
func (k Kind) Trustable() bool {
	return k == KindFormula || k == KindCask || k == KindTap
}

func (k Kind) flag() string { return "--" + string(k) }

// Add records names in the Brewfile and installs the bundle. Tap-qualified
// names of trustable kinds go through the `brew trust` flow first: when one
// is untrusted, confirmTrust decides whether to trust it; declining aborts
// before anything is written.
func Add(ctx context.Context, r Runner, path string, kind Kind, names []string, confirmTrust func(name string) (bool, error)) error {
	for _, name := range names {
		if !NeedsTrust(name) || !kind.Trustable() {
			continue
		}
		trusted, err := IsTrusted(ctx, r, kind, name)
		if err != nil {
			return err
		}
		if trusted {
			continue
		}
		ok, err := confirmTrust(name)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s %q is not trusted (declined)", kind, name)
		}
		if err := Trust(ctx, r, kind, name); err != nil {
			return err
		}
	}

	addArgs := append([]string{"bundle", "add", "--file=" + path, kind.flag()}, names...)
	if err := r.Run(ctx, "brew", addArgs...); err != nil {
		return err
	}
	return r.Run(ctx, "brew", "bundle", "install", "--file="+path)
}

// Upgrade installs and upgrades everything in the Brewfile without removing
// anything.
func Upgrade(ctx context.Context, r Runner, path string) error {
	return r.Run(ctx, "brew", "bundle", "install", "--file="+path, "--upgrade", "--no-cleanup")
}

// Sync makes the machine match the Brewfile exactly, removing brews that are
// not listed. Unless force is set, a cleanup dry-run lists the would-be
// removals first and confirm decides whether to proceed; returning false
// aborts with no changes.
func Sync(ctx context.Context, r Runner, path string, force bool, confirm func(removals []string) (bool, error)) error {
	if !force {
		removals, err := cleanupDryRun(ctx, r, path)
		if err != nil {
			return err
		}
		if len(removals) > 0 {
			ok, err := confirm(removals)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}
	}
	return r.Run(ctx, "brew", "bundle", "install", "--file="+path,
		"--force", "--force-cleanup", "--upgrade", "--zap")
}

// cleanupDryRun runs `brew bundle cleanup` without --force, which exits
// non-zero and prints the would-be removals when anything would go. A failure
// with no stdout is a real error, not a removal listing.
func cleanupDryRun(ctx context.Context, r Runner, path string) ([]string, error) {
	out, err := r.Output(ctx, "brew", "bundle", "cleanup", "--file="+path)
	if err == nil {
		return nil, nil
	}
	lines := nonEmptyLines(out)
	if len(lines) == 0 {
		return nil, fmt.Errorf("check for removable brews: %w", err)
	}
	return lines, nil
}

// dumpKinds is what `dotty brewfile dump` writes by default, per DESIGN.
var dumpKinds = []Kind{KindFormula, KindCask, "mas", KindFlatpak}

// allDumpKinds is every positive type flag `brew bundle dump` accepts —
// Homebrew has no --all flag, so dotty's --all spells them out.
var allDumpKinds = []Kind{KindFormula, KindCask, KindTap, "mas", KindFlatpak,
	KindVSCode, KindGo, KindCargo, KindUV, KindKrew, KindNPM}

// Dump snapshots the installed brews into the Brewfile. force overwrites an
// existing file; deciding whether that is wanted is the caller's job.
func Dump(ctx context.Context, r Runner, path string, all, force bool) error {
	args := []string{"bundle", "dump", "--file=" + path}
	kinds := dumpKinds
	if all {
		kinds = allDumpKinds
	}
	for _, k := range kinds {
		args = append(args, k.flag())
	}
	if force {
		args = append(args, "--force")
	}
	return r.Run(ctx, "brew", args...)
}

func nonEmptyLines(out []byte) []string {
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
