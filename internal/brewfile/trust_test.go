// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package brewfile

import (
	"context"
	"testing"
)

func TestNeedsTrust(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "jq", want: false},
		{name: "fluxcd/tap", want: false}, // one slash: a plain tap name
		{name: "acme/tap/widget", want: true},
		{name: "a/b/c/d", want: true},
		{name: "", want: false},
	}
	for _, tt := range tests {
		if got := NeedsTrust(tt.name); got != tt.want {
			t.Errorf("NeedsTrust(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsTrusted(t *testing.T) {
	// Shape captured from a live `brew trust --json v1` on Homebrew 6.0.0.
	doc := `{
	  "taps": ["fluxcd/tap", "hashicorp/tap"],
	  "formulae": ["anomalyco/tap/opencode", "derailed/k9s/k9s"],
	  "casks": ["terraform-linters/tap/tflint"],
	  "commands": []
	}`
	tests := []struct {
		name    string
		kind    Kind
		target  string
		want    bool
		wantErr bool
	}{
		{name: "trusted formula", kind: KindFormula, target: "anomalyco/tap/opencode", want: true},
		{name: "untrusted formula", kind: KindFormula, target: "acme/tap/widget", want: false},
		{name: "trusted cask", kind: KindCask, target: "terraform-linters/tap/tflint", want: true},
		{name: "trusted tap", kind: KindTap, target: "fluxcd/tap", want: true},
		{name: "cask name not valid as formula", kind: KindFormula, target: "terraform-linters/tap/tflint", want: false},
		{name: "untrustable kind errors", kind: KindNPM, target: "x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeRunner{outputs: [][]byte{[]byte(doc)}}
			got, err := IsTrusted(context.Background(), r, tt.kind, tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("IsTrusted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrustRejectsUntrustableKinds(t *testing.T) {
	r := &fakeRunner{}
	if err := Trust(context.Background(), r, KindGo, "x"); err == nil {
		t.Error("Trust(go) error = nil, want failure")
	}
	if len(r.calls) != 0 {
		t.Errorf("calls = %v, want none", r.calls)
	}
}

func FuzzDecodeTrustList(f *testing.F) {
	f.Add(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`)
	f.Add(`{"taps":["a/b"],"formulae":["a/b/c"]}`)
	f.Add(`not json at all`)
	f.Add(`[]`)
	f.Fuzz(func(t *testing.T, data string) {
		// External-tool output must never panic the decoder; errors are fine.
		_, _ = decodeTrustList([]byte(data))
	})
}
