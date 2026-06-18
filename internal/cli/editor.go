// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// interactiveRunner is the slice of ExecRunner the editor helpers consume;
// tests substitute a fake that writes into the file it is handed.
type interactiveRunner interface {
	RunInteractive(ctx context.Context, name string, args ...string) error
}

// EditorCommand resolves the user's editor: $VISUAL, then $EDITOR, then vi.
// The value is split on whitespace so entries like "code --wait" work.
func EditorCommand() (name string, args []string) {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			fields := strings.Fields(v)
			return fields[0], fields[1:]
		}
	}
	return "vi", nil
}

// EditFile opens path in the user's editor and waits for it to exit.
func EditFile(ctx context.Context, r interactiveRunner, path string) error {
	name, args := EditorCommand()
	if err := r.RunInteractive(ctx, name, append(args, path)...); err != nil {
		return fmt.Errorf("edit %s: %w", path, err)
	}
	return nil
}

// EditTempFile seeds a temp file with initial, opens it in the user's editor,
// and returns the edited content with surrounding whitespace trimmed. The
// temp file is always removed.
func EditTempFile(ctx context.Context, r interactiveRunner, initial string) (string, error) {
	tmp, err := os.CreateTemp("", "dotty-edit-*.md")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	path := tmp.Name()
	defer os.Remove(path)

	if _, err := tmp.WriteString(initial); err != nil {
		tmp.Close()
		return "", fmt.Errorf("seed %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close %s: %w", path, err)
	}

	if err := EditFile(ctx, r, path); err != nil {
		return "", err
	}

	edited, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s back: %w", filepath.Base(path), err)
	}
	return strings.TrimSpace(string(edited)), nil
}
