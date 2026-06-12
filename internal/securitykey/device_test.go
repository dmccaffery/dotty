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
	"context"
	"errors"
	"slices"
	"testing"
)

// fakeRunner serves canned Output results keyed by the invoked program.
type fakeRunner struct {
	byProgram map[string]struct {
		out []byte
		err error
	}
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	r, ok := f.byProgram[name]
	if !ok {
		return nil, errors.New("unexpected program " + name)
	}
	return r.out, r.err
}

func newFake(ykmanOut string, fidoOut string) *fakeRunner {
	return &fakeRunner{byProgram: map[string]struct {
		out []byte
		err error
	}{
		"ykman":       {out: []byte(ykmanOut)},
		"fido2-token": {out: []byte(fidoOut)},
	}}
}

func TestListSerials(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		want    []string
		wantErr bool
	}{
		{name: "two keys", out: "12345678\n87654321\n", want: []string{"12345678", "87654321"}},
		{name: "single key no trailing newline", out: "12345678", want: []string{"12345678"}},
		{name: "no keys", out: "", want: nil},
		{name: "blank lines tolerated", out: "\n12345678\n\n", want: []string{"12345678"}},
		{name: "junk rejected", out: "WARNING: foo\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListSerials(context.Background(), newFake(tt.out, ""))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("serials = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListFIDODevices(t *testing.T) {
	// Format captured live from fido2-token 1.17.0 on darwin.
	out := "ioreg://4295277255: vendor=0x1050, product=0x0407 (Yubico YubiKey OTP+FIDO+CCID)\n" +
		"ioreg://4295277299: vendor=0x32a3, product=0x3201 (SoloKeys Solo 2)\n" +
		"garbage line that matches nothing\n"
	devices, err := ListFIDODevices(context.Background(), newFake("", out))
	if err != nil {
		t.Fatalf("ListFIDODevices() error: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("devices = %+v, want 2", devices)
	}
	if devices[0].Path != "ioreg://4295277255" || devices[0].Vendor != "1050" {
		t.Errorf("device 0 = %+v", devices[0])
	}
	if devices[0].Label != "Yubico YubiKey OTP+FIDO+CCID" {
		t.Errorf("label = %q", devices[0].Label)
	}

	paths := YubicoPaths(devices)
	if !slices.Equal(paths, []string{"ioreg://4295277255"}) {
		t.Errorf("YubicoPaths = %v, want only the Yubico device", paths)
	}
}
