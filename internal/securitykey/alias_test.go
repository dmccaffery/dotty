// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package securitykey

import "testing"

func TestValidateName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{input: "work", wantErr: false},
		{input: "Work-2.bak_1", wantErr: false},
		{input: "", wantErr: true},
		{input: "123", wantErr: true},      // all digits would shadow a serial
		{input: "1work", wantErr: true},    // must start with a letter
		{input: "wo rk", wantErr: true},    // no spaces
		{input: "work/key", wantErr: true}, // no separators
		{input: "-work", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestIsSerial(t *testing.T) {
	for input, want := range map[string]bool{
		"12345678": true,
		"0":        true,
		"":         false,
		"12a4":     false,
		"work":     false,
	} {
		if got := IsSerial(input); got != want {
			t.Errorf("IsSerial(%q) = %v, want %v", input, got, want)
		}
	}
}

func FuzzValidateName(f *testing.F) {
	f.Add("work")
	f.Add("123")
	f.Add("Δkey")
	f.Add(" leading")
	f.Fuzz(func(t *testing.T, name string) {
		// Invariant: a name that validates can never be mistaken for a serial,
		// so --security-key=<ref> resolution is never ambiguous.
		if err := ValidateName(name); err == nil && IsSerial(name) {
			t.Errorf("%q passes ValidateName and IsSerial", name)
		}
	})
}
