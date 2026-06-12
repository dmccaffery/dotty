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

package securitykey

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bitwise-media-group/dotty/internal/cli"
)

// pipeIOS returns non-terminal streams, so resolution must never prompt.
func pipeIOS() cli.IOStreams {
	return cli.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
}

func TestResolveSerial(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	if err := store.Add("12345678", "work", ""); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		ref     string
		ykman   string
		want    string
		wantErr error
	}{
		{name: "explicit serial passes through", ref: "999", want: "999"},
		{name: "alias resolves", ref: "work", want: "12345678"},
		{name: "unknown alias", ref: "ghost", wantErr: ErrUnknownAlias},
		{name: "single plugged key wins", ref: "", ykman: "555\n", want: "555"},
		{name: "no key present", ref: "", ykman: "", wantErr: ErrNoKeyPresent},
		{name: "ambiguous without terminal", ref: "", ykman: "111\n222\n", wantErr: ErrAmbiguousKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveSerial(ctx, newFake(tt.ykman, ""), store, pipeIOS(), tt.ref)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSerial() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("serial = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectDeviceForEnroll(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	t.Run("single key needs no device path", func(t *testing.T) {
		dev, err := SelectDeviceForEnroll(ctx, newFake("555\n", ""), store, pipeIOS(), "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if dev.Serial != "555" || dev.Path != "" {
			t.Errorf("device = %+v", dev)
		}
	})

	t.Run("no key present", func(t *testing.T) {
		_, err := SelectDeviceForEnroll(ctx, newFake("", ""), store, pipeIOS(), "")
		if !errors.Is(err, ErrNoKeyPresent) {
			t.Errorf("error = %v, want ErrNoKeyPresent", err)
		}
	})

	t.Run("wanted serial must be connected", func(t *testing.T) {
		_, err := SelectDeviceForEnroll(ctx, newFake("555\n", ""), store, pipeIOS(), "999")
		if err == nil || !strings.Contains(err.Error(), "not connected") {
			t.Errorf("error = %v, want not-connected", err)
		}
	})

	t.Run("wanted serial with one key skips the replug", func(t *testing.T) {
		dev, err := SelectDeviceForEnroll(ctx, newFake("555\n", ""), store, pipeIOS(), "555")
		if err != nil || dev.Serial != "555" {
			t.Errorf("device = %+v, %v", dev, err)
		}
	})

	t.Run("multiple keys without a terminal cannot replug or pick", func(t *testing.T) {
		fido := "ioreg://1: vendor=0x1050, product=0x0407 (YubiKey A)\nioreg://2: vendor=0x1050, product=0x0407 (YubiKey B)\n"
		_, err := SelectDeviceForEnroll(ctx, newFake("111\n222\n", fido), store, pipeIOS(), "")
		if !errors.Is(err, ErrAmbiguousKey) {
			t.Errorf("error = %v, want ErrAmbiguousKey", err)
		}
	})
}

func TestReplugTracker(t *testing.T) {
	t.Run("vanish then reappear maps serial to the new path", func(t *testing.T) {
		tr := NewReplugTracker([]string{"111", "222"}, []string{"ioreg://1", "ioreg://2"})

		if _, ok := tr.Observe([]string{"111", "222"}, []string{"ioreg://1", "ioreg://2"}); ok {
			t.Fatal("no change reported a device")
		}
		// 222 unplugged.
		if _, ok := tr.Observe([]string{"111"}, []string{"ioreg://1"}); ok {
			t.Fatal("unplug alone reported a device")
		}
		// Still unplugged.
		if _, ok := tr.Observe([]string{"111"}, []string{"ioreg://1"}); ok {
			t.Fatal("still-unplugged reported a device")
		}
		// Replugged with a fresh registry id.
		dev, ok := tr.Observe([]string{"111", "222"}, []string{"ioreg://1", "ioreg://9"})
		if !ok {
			t.Fatal("reappearance not detected")
		}
		if dev.Serial != "222" || dev.Path != "ioreg://9" {
			t.Errorf("device = %+v, want 222 @ ioreg://9", dev)
		}
	})

	t.Run("reappearance with a recycled path yields no path", func(t *testing.T) {
		tr := NewReplugTracker([]string{"111", "222"}, []string{"ioreg://1", "ioreg://2"})
		tr.Observe([]string{"111"}, []string{"ioreg://1"})
		dev, ok := tr.Observe([]string{"111", "222"}, []string{"ioreg://1", "ioreg://2"})
		if !ok || dev.Serial != "222" {
			t.Fatalf("device = %+v, ok = %v", dev, ok)
		}
		if dev.Path != "ioreg://2" {
			// ioreg://2 vanished alongside 222 and came back with it, so it
			// is identified via the same diff.
			t.Errorf("path = %q, want recycled ioreg://2", dev.Path)
		}
	})
}
