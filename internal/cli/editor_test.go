// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestEditorCommand(t *testing.T) {
	tests := []struct {
		name     string
		visual   string
		editor   string
		wantName string
		wantArgs int
	}{
		{name: "VISUAL wins over EDITOR", visual: "code --wait", editor: "vim", wantName: "code", wantArgs: 1},
		{name: "EDITOR when VISUAL unset", visual: "", editor: "nano", wantName: "nano", wantArgs: 0},
		{name: "vi fallback", visual: "", editor: "", wantName: "vi", wantArgs: 0},
		{name: "whitespace-only values are ignored", visual: "  ", editor: " ", wantName: "vi", wantArgs: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)
			name, args := EditorCommand()
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if len(args) != tt.wantArgs {
				t.Errorf("args = %v, want %d args", args, tt.wantArgs)
			}
		})
	}
}

// fakeEditor simulates an editor by writing content into the file passed as
// the final argument.
type fakeEditor struct {
	content string
	err     error
	gotName string
}

func (f *fakeEditor) RunInteractive(_ context.Context, name string, args ...string) error {
	f.gotName = name
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(args[len(args)-1], []byte(f.content), 0o600)
}

func TestEditTempFile(t *testing.T) {
	t.Setenv("VISUAL", "fake-editor")
	t.Setenv("EDITOR", "")

	t.Run("returns trimmed edited content", func(t *testing.T) {
		ed := &fakeEditor{content: "  a description\n\n"}
		got, err := EditTempFile(context.Background(), ed, "seed text")
		if err != nil {
			t.Fatalf("EditTempFile() error: %v", err)
		}
		if got != "a description" {
			t.Errorf("content = %q, want %q", got, "a description")
		}
		if ed.gotName != "fake-editor" {
			t.Errorf("editor invoked = %q, want fake-editor", ed.gotName)
		}
	})

	t.Run("editor failure surfaces", func(t *testing.T) {
		ed := &fakeEditor{err: errors.New("editor exploded")}
		if _, err := EditTempFile(context.Background(), ed, ""); err == nil {
			t.Fatal("EditTempFile() error = nil, want editor failure")
		}
	})
}
