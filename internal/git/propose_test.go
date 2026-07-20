// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// autoMergeFakeRunner cans Output results keyed by the full command line
// ("name arg arg ..."), recording every invocation for order/absence checks.
type autoMergeFakeRunner struct {
	outputs map[string]string
	errs    map[string]error
	calls   []string
}

func (f *autoMergeFakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	if err, ok := f.errs[key]; ok {
		return nil, err
	}
	out, ok := f.outputs[key]
	if !ok {
		return nil, fmt.Errorf("unexpected command %q", key)
	}
	return []byte(out), nil
}

func (f *autoMergeFakeRunner) Run(context.Context, string, ...string) error            { return nil }
func (f *autoMergeFakeRunner) RunInteractive(context.Context, string, ...string) error { return nil }

const (
	fakeRemoteURLCmd = "git remote get-url upstream"
	fakeRepoViewCmd  = "gh repo view acme/widgets --json " +
		"autoMergeAllowed,rebaseMergeAllowed,squashMergeAllowed,mergeCommitAllowed"
)

func TestCheckAutoMerge(t *testing.T) {
	const allAllowed = `{"autoMergeAllowed":true,"rebaseMergeAllowed":true,` +
		`"squashMergeAllowed":true,"mergeCommitAllowed":true}`
	cases := []struct {
		name       string
		repoJSON   string
		method     string
		wantIs     error  // matched with errors.Is when set
		wantErrSub string // substring match when set; both empty means success
	}{
		{"rebase allowed", allAllowed, "rebase", nil, ""},
		{"squash allowed", allAllowed, "squash", nil, ""},
		{"merge commit allowed", allAllowed, "merge", nil, ""},
		{
			name:     "auto-merge disabled is the sentinel",
			repoJSON: `{"autoMergeAllowed":false,"rebaseMergeAllowed":true,"squashMergeAllowed":true,"mergeCommitAllowed":true}`,
			method:   "rebase",
			wantIs:   ErrAutoMergeUnavailable,
		},
		{
			name: "disallowed method names the method",
			repoJSON: `{"autoMergeAllowed":true,"rebaseMergeAllowed":false,` +
				`"squashMergeAllowed":true,"mergeCommitAllowed":true}`,
			method:     "rebase",
			wantErrSub: "rebase",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &autoMergeFakeRunner{outputs: map[string]string{
				fakeRemoteURLCmd: "https://github.com/acme/widgets.git\n",
				fakeRepoViewCmd:  c.repoJSON,
			}}
			err := CheckAutoMerge(context.Background(), fake, "upstream", c.method)
			switch {
			case c.wantIs != nil:
				if !errors.Is(err, c.wantIs) {
					t.Fatalf("CheckAutoMerge(%q) error = %v, want %v", c.method, err, c.wantIs)
				}
			case c.wantErrSub != "":
				if err == nil || !strings.Contains(err.Error(), c.wantErrSub) {
					t.Fatalf("CheckAutoMerge(%q) error = %v, want substring %q", c.method, err, c.wantErrSub)
				}
			case err != nil:
				t.Fatalf("CheckAutoMerge(%q) error: %v", c.method, err)
			}
		})
	}
}

func TestEnableAutoMerge(t *testing.T) {
	const (
		viewCmd  = "gh pr view 7 --repo acme/widgets --json autoMergeRequest --jq .autoMergeRequest"
		mergeCmd = "gh pr merge 7 --repo acme/widgets --auto --rebase"
	)
	cases := []struct {
		name        string
		pending     string
		mergeErr    error
		wantAlready bool
		wantErr     bool
		wantMerged  bool
	}{
		{"enables when no request pending", "null\n", nil, false, false, true},
		{"skips a pending request", `{"enabledAt":"2026-07-20T00:00:00Z"}` + "\n", nil, true, false, false},
		{"surfaces the gh failure", "null\n", errors.New("auto merge is not allowed"), false, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &autoMergeFakeRunner{
				outputs: map[string]string{
					fakeRemoteURLCmd: "https://github.com/acme/widgets.git\n",
					viewCmd:          c.pending,
					mergeCmd:         "",
				},
			}
			if c.mergeErr != nil {
				fake.errs = map[string]error{mergeCmd: c.mergeErr}
			}
			already, err := EnableAutoMerge(context.Background(), fake, "upstream", 7, "rebase")
			if (err != nil) != c.wantErr {
				t.Fatalf("EnableAutoMerge() error = %v, wantErr %v", err, c.wantErr)
			}
			if already != c.wantAlready {
				t.Errorf("EnableAutoMerge() already = %v, want %v", already, c.wantAlready)
			}
			if merged := slices.Contains(fake.calls, mergeCmd); merged != c.wantMerged {
				t.Errorf("gh pr merge invoked = %v, want %v", merged, c.wantMerged)
			}
		})
	}
}

func TestAddAutoMergeComment(t *testing.T) {
	const (
		viewCmd    = "gh pr view 7 --repo acme/widgets --json comments --jq .comments[].body"
		commentCmd = "gh pr comment 7 --repo acme/widgets --body /auto-merge"
	)
	cases := []struct {
		name       string
		comments   string
		wantAdded  bool
		wantPosted bool
	}{
		{"posts on a PR without the comment", "looks good\n", true, true},
		{"posts when there are no comments", "", true, true},
		{"skips a PR already carrying the comment", "looks good\n/auto-merge\n", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &autoMergeFakeRunner{outputs: map[string]string{
				fakeRemoteURLCmd: "https://github.com/acme/widgets.git\n",
				viewCmd:          c.comments,
				commentCmd:       "",
			}}
			added, err := AddAutoMergeComment(context.Background(), fake, "upstream", 7)
			if err != nil {
				t.Fatalf("AddAutoMergeComment() error: %v", err)
			}
			if added != c.wantAdded {
				t.Errorf("AddAutoMergeComment() added = %v, want %v", added, c.wantAdded)
			}
			if posted := slices.Contains(fake.calls, commentCmd); posted != c.wantPosted {
				t.Errorf("gh pr comment invoked = %v, want %v", posted, c.wantPosted)
			}
		})
	}
}
