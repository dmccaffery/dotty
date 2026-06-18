// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package env

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCapture(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantOut     string
		wantSecrets map[string]string
	}{
		{
			name:        "simple assignment",
			src:         "TOKEN=abc123",
			wantOut:     "TOKEN={{ dotty://ci/TOKEN }}",
			wantSecrets: map[string]string{"TOKEN": "abc123"},
		},
		{
			name:        "export prefix and spacing preserved",
			src:         "export  AWS_KEY = secret",
			wantOut:     "export  AWS_KEY = {{ dotty://ci/AWS_KEY }}",
			wantSecrets: map[string]string{"AWS_KEY": "secret"},
		},
		{
			name:        "indentation preserved",
			src:         "\tNESTED=value",
			wantOut:     "\tNESTED={{ dotty://ci/NESTED }}",
			wantSecrets: map[string]string{"NESTED": "value"},
		},
		{
			name:        "double quotes are decoded",
			src:         `MSG="line1\nline2"`,
			wantOut:     "MSG={{ dotty://ci/MSG }}",
			wantSecrets: map[string]string{"MSG": "line1\nline2"},
		},
		{
			name:        "single quotes are literal",
			src:         `RAW='a\nb'`,
			wantOut:     "RAW={{ dotty://ci/RAW }}",
			wantSecrets: map[string]string{"RAW": `a\nb`},
		},
		{
			name:        "inline comment preserved",
			src:         "KEY=value # trailing note",
			wantOut:     "KEY={{ dotty://ci/KEY }} # trailing note",
			wantSecrets: map[string]string{"KEY": "value"},
		},
		{
			name:        "hash without leading space stays in value",
			src:         "PASS=p#ss",
			wantOut:     "PASS={{ dotty://ci/PASS }}",
			wantSecrets: map[string]string{"PASS": "p#ss"},
		},
		{
			name:        "comment after quoted value preserved",
			src:         `Q="v" # note`,
			wantOut:     "Q={{ dotty://ci/Q }} # note",
			wantSecrets: map[string]string{"Q": "v"},
		},
		{
			name:        "value with equals sign",
			src:         "PAIR=a=b=c",
			wantOut:     "PAIR={{ dotty://ci/PAIR }}",
			wantSecrets: map[string]string{"PAIR": "a=b=c"},
		},
		{
			name:        "comments and blanks untouched",
			src:         "# a comment\n\nKEY=v\n",
			wantOut:     "# a comment\n\nKEY={{ dotty://ci/KEY }}\n",
			wantSecrets: map[string]string{"KEY": "v"},
		},
		{
			name:        "empty value left as is",
			src:         "EMPTY=\nQUOTED_EMPTY=\"\"",
			wantOut:     "EMPTY=\nQUOTED_EMPTY=\"\"",
			wantSecrets: map[string]string{},
		},
		{
			name:        "existing reference is idempotent",
			src:         "TOKEN={{ dotty://ci/TOKEN }}",
			wantOut:     "TOKEN={{ dotty://ci/TOKEN }}",
			wantSecrets: map[string]string{},
		},
		{
			name:        "non-assignment lines passthrough",
			src:         "not an assignment\n1BAD=x",
			wantOut:     "not an assignment\n1BAD=x",
			wantSecrets: map[string]string{},
		},
		{
			name:        "unterminated quote left untouched",
			src:         `OPEN="no close`,
			wantOut:     `OPEN="no close`,
			wantSecrets: map[string]string{},
		},
		{
			name:        "crlf endings preserved",
			src:         "A=1\r\nB=2\r\n",
			wantOut:     "A={{ dotty://ci/A }}\r\nB={{ dotty://ci/B }}\r\n",
			wantSecrets: map[string]string{"A": "1", "B": "2"},
		},
		{
			name:        "last duplicate wins",
			src:         "DUP=first\nDUP=second",
			wantOut:     "DUP={{ dotty://ci/DUP }}\nDUP={{ dotty://ci/DUP }}",
			wantSecrets: map[string]string{"DUP": "second"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, secrets, err := Capture(tt.src, "ci")
			if err != nil {
				t.Fatalf("Capture(%q): %v", tt.src, err)
			}
			if out != tt.wantOut {
				t.Errorf("Capture(%q) out =\n%q\nwant\n%q", tt.src, out, tt.wantOut)
			}
			if !reflect.DeepEqual(secrets, tt.wantSecrets) {
				t.Errorf("Capture(%q) secrets = %v, want %v", tt.src, secrets, tt.wantSecrets)
			}
		})
	}
}

func TestCaptureInvalidNamespace(t *testing.T) {
	if _, _, err := Capture("KEY=v", "bad:ns"); err == nil {
		t.Fatal("Capture with invalid namespace returned nil error")
	}
}

// TestCaptureInjectRoundTrip checks that Capture and Inject are inverses: the
// references Capture writes resolve, via the secrets it collected, back to the
// source. Values are unquoted here so decoding is the identity and the text
// round-trips exactly; quoted values intentionally decode to their literal
// secret (covered by TestCaptureDecodesQuotedValue).
func TestCaptureInjectRoundTrip(t *testing.T) {
	src := "export A=alpha\nB = two-words # note\nC=literal\n"
	out, secrets, err := Capture(src, "ns")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	resolved, err := Inject(out, func(ns, key string) (string, error) {
		if ns != "ns" {
			t.Errorf("resolver got namespace %q, want ns", ns)
		}
		return secrets[key], nil
	})
	if err != nil {
		t.Fatalf("Inject: %v", err)
	}
	if resolved != src {
		t.Errorf("round trip = %q, want %q", resolved, src)
	}
}

// TestCaptureDecodesQuotedValue documents that the stored secret is the decoded
// value: .env quoting and escapes are stripped so the keychain holds the real
// value that env run / env use will emit.
func TestCaptureDecodesQuotedValue(t *testing.T) {
	_, secrets, err := Capture(`B="two\twords"`, "ns")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if got := secrets["B"]; got != "two\twords" {
		t.Errorf("stored value = %q, want %q", got, "two\twords")
	}
}

// TestCaptureIdempotent checks that capturing already-captured output is a
// no-op: the text is unchanged and nothing new is collected.
func TestCaptureIdempotent(t *testing.T) {
	src := "A=1\nB=2\n"
	once, _, err := Capture(src, "ns")
	if err != nil {
		t.Fatal(err)
	}
	twice, secrets, err := Capture(once, "ns")
	if err != nil {
		t.Fatal(err)
	}
	if twice != once {
		t.Errorf("second Capture = %q, want unchanged %q", twice, once)
	}
	if len(secrets) != 0 {
		t.Errorf("second Capture collected %v, want none", secrets)
	}
}

// testResolve is the resolver the Parse tests inject references through. It
// maps "<namespace>/<key>" to a value, defaulting an empty namespace (the bare
// {{ KEY }} form) to "default", and reports ErrKeyNotFound for anything else.
func testResolve(ns, key string) (string, error) {
	if ns == "" {
		ns = "default"
	}
	secrets := map[string]string{
		"prod/API_KEY":  "live-key",
		"prod/HOST":     "db.internal",
		"prod/MULTI":    "a#b\nc",
		"default/TOKEN": "tok",
	}
	if v, ok := secrets[ns+"/"+key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("%q in namespace %q: %w", key, ns, ErrKeyNotFound)
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Entry
	}{
		{
			name: "literal passthrough",
			src:  "LOG_LEVEL=debug",
			want: []Entry{{"LOG_LEVEL", "debug"}},
		},
		{
			name: "fully-qualified reference resolved",
			src:  "API_KEY={{ dotty://prod/API_KEY }}",
			want: []Entry{{"API_KEY", "live-key"}},
		},
		{
			name: "bare reference falls back to namespace",
			src:  "TOKEN={{ TOKEN }}",
			want: []Entry{{"TOKEN", "tok"}},
		},
		{
			name: "reference embedded mid-value",
			src:  "DSN=postgres://{{ dotty://prod/HOST }}/db",
			want: []Entry{{"DSN", "postgres://db.internal/db"}},
		},
		{
			name: "resolved secret keeps special characters",
			src:  "MULTI={{ dotty://prod/MULTI }}",
			want: []Entry{{"MULTI", "a#b\nc"}},
		},
		{
			name: "mix of secrets and plain vars",
			src:  "# a comment\n\nPORT=8080\nAPI_KEY={{ dotty://prod/API_KEY }}\n",
			want: []Entry{{"PORT", "8080"}, {"API_KEY", "live-key"}},
		},
		{
			name: "double quotes decoded",
			src:  `MSG="line1\nline2"`,
			want: []Entry{{"MSG", "line1\nline2"}},
		},
		{
			name: "single quotes literal",
			src:  `RAW='a\nb'`,
			want: []Entry{{"RAW", `a\nb`}},
		},
		{
			name: "inline comment stripped",
			src:  "KEY=value # trailing note",
			want: []Entry{{"KEY", "value"}},
		},
		{
			name: "hash without leading space kept",
			src:  "PASS=p#ss",
			want: []Entry{{"PASS", "p#ss"}},
		},
		{
			name: "export prefix and spacing",
			src:  "export  AWS_KEY = secret",
			want: []Entry{{"AWS_KEY", "secret"}},
		},
		{
			name: "empty value is kept",
			src:  "EMPTY=\nQUOTED_EMPTY=\"\"",
			want: []Entry{{"EMPTY", ""}, {"QUOTED_EMPTY", ""}},
		},
		{
			name: "non-assignment lines skipped",
			src:  "not an assignment\n1BAD=x\nGOOD=ok",
			want: []Entry{{"GOOD", "ok"}},
		},
		{
			name: "crlf endings",
			src:  "A=1\r\nB=2\r\n",
			want: []Entry{{"A", "1"}, {"B", "2"}},
		},
		{
			name: "duplicate keys keep file order",
			src:  "DUP=first\nDUP=second",
			want: []Entry{{"DUP", "first"}, {"DUP", "second"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.src, testResolve)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.src, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) =\n%#v\nwant\n%#v", tt.src, got, tt.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantSub string
	}{
		{
			name:    "unterminated quote",
			src:     `OPEN="no close`,
			wantSub: "unterminated quote",
		},
		{
			name:    "unknown reference",
			src:     "X={{ dotty://prod/MISSING }}",
			wantSub: "key not found",
		},
		{
			name:    "malformed reference",
			src:     "X={{ dotty://prod }}",
			wantSub: "malformed reference",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.src, testResolve)
			if err == nil {
				t.Fatalf("Parse(%q) = nil error, want one containing %q", tt.src, tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("Parse(%q) error = %q, want substring %q", tt.src, err, tt.wantSub)
			}
		})
	}
}

// TestCaptureParseRoundTrip checks that Parse loads what Capture externalized:
// capturing a .env collects its secrets and rewrites values as references, and
// parsing that output back — resolving through the collected secrets — yields
// every assignment with its original value.
func TestCaptureParseRoundTrip(t *testing.T) {
	src := "export A=alpha\nPORT=8080\nB = two-words # note\n"
	out, secrets, err := Capture(src, "ns")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	got, err := Parse(out, func(ns, key string) (string, error) {
		if ns != "ns" {
			t.Errorf("resolver got namespace %q, want ns", ns)
		}
		v, ok := secrets[key]
		if !ok {
			return "", fmt.Errorf("%q: %w", key, ErrKeyNotFound)
		}
		return v, nil
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []Entry{{"A", "alpha"}, {"PORT", "8080"}, {"B", "two-words"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %#v, want %#v", got, want)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"K=v", "export K = v # c", `Q="a\nb"`, "K={{ dotty://ns/K }}",
		"# comment", "", "K=", "1BAD=x", "A=1\r\nB=2", `OPEN="no close`,
		"BARE={{ K }}", "MID=a{{ dotty://ns/K }}b",
	} {
		f.Add(seed)
	}
	resolve := func(ns, key string) (string, error) { return "x", nil }
	f.Fuzz(func(t *testing.T, src string) {
		entries, err := Parse(src, resolve) // must never panic
		if err != nil {
			return
		}
		// Every returned key is a valid environment variable name, so the pairs
		// are safe to hand to os/exec.
		for _, e := range entries {
			if err := ValidateKey(e.Key); err != nil {
				t.Errorf("Parse(%q) returned invalid key %q: %v", src, e.Key, err)
			}
		}
	})
}

func FuzzCapture(f *testing.F) {
	for _, seed := range []string{
		"K=v", "export K = v # c", `Q="a\nb"`, "K={{ dotty://ns/K }}",
		"# comment", "", "K=", "1BAD=x", "A=1\r\nB=2", `OPEN="no close`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		out, secrets, err := Capture(src, "ns") // must never panic
		if err != nil {
			return
		}
		// Every captured key is rewritten to its reference in the output.
		for key := range secrets {
			ref := "{{ dotty://ns/" + key + " }}"
			if !strings.Contains(out, ref) {
				t.Errorf("Capture(%q) captured %q but output %q lacks %q", src, key, out, ref)
			}
		}
		// Capturing the output again is a no-op: references are recognized and
		// left in place, so nothing new is collected.
		again, more, err := Capture(out, "ns")
		if err != nil {
			t.Fatalf("second Capture(%q): %v", out, err)
		}
		if again != out {
			t.Errorf("Capture not idempotent: %q -> %q", out, again)
		}
		if len(more) != 0 {
			t.Errorf("second Capture collected %v, want none", more)
		}
	})
}
