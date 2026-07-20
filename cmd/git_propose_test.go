// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import "testing"

func TestAutoMergeModeSet(t *testing.T) {
	cases := []struct {
		in      string
		want    autoMergeMode
		wantErr bool
	}{
		{"merge", "merge", false},
		{"rebase", "rebase", false},
		{"squash", "squash", false},
		{"comment", autoMergeComment, false},
		{"REBASE", "rebase", false},
		{"COMMENT", autoMergeComment, false},
		{"true", "", true},
		{"false", "", true},
		{"yes", "", true},
		{"banana", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		t.Run("input "+c.in, func(t *testing.T) {
			var m autoMergeMode
			err := m.Set(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("Set(%q) error = %v, wantErr %v", c.in, err, c.wantErr)
			}
			if !c.wantErr && m != c.want {
				t.Errorf("Set(%q) = %q, want %q", c.in, m, c.want)
			}
		})
	}
}

func TestAutoMergeModeMergeMethod(t *testing.T) {
	cases := []struct {
		mode autoMergeMode
		want string
	}{
		{autoMergeOff, ""},
		{autoMergeComment, ""},
		{"merge", "merge"},
		{"rebase", "rebase"},
		{"squash", "squash"},
	}
	for _, c := range cases {
		if got := c.mode.mergeMethod(); got != c.want {
			t.Errorf("(%q).mergeMethod() = %q, want %q", c.mode, got, c.want)
		}
	}
}
