package main

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "removes format control",
			in:   "Foo\u202ABar",
			out:  "FooBar",
		},
		{
			name: "removes ascii control",
			in:   "Hello\u0007World",
			out:  "HelloWorld",
		},
		{
			name: "normalizes whitespace",
			in:   "  App\tName\n",
			out:  "App Name",
		},
		{
			name: "empty remains empty",
			in:   "",
			out:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitize(tc.in); got != tc.out {
				t.Fatalf("sanitize(%q) = %q, want %q", tc.in, got, tc.out)
			}
		})
	}
}

func TestProcessCSVOutput(t *testing.T) {
	originalResolve := resolveFunc
	defer func() {
		resolveFunc = originalResolve
	}()

	resolveFunc = func(id string) (record, error) {
		if id != "123" {
			t.Fatalf("unexpected id: %s", id)
		}
		return record{
			Bundle:    id,
			Name:      "My,App",
			Publisher: "Dev",
			URL:       "https://example.com/app",
		}, nil
	}

	input := strings.NewReader("123\n")
	var out strings.Builder
	fields := []Field{FieldBundle, FieldName, FieldPublisher, FieldURL}

	if err := process(input, &out, fields, true, false, true); err != nil {
		t.Fatalf("process returned error: %v", err)
	}

	got := out.String()
	want := "bundle,name,publisher,url\n123,\"My,App\",Dev,https://example.com/app\n"
	if got != want {
		t.Fatalf("csv output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestProcessTSVOutput(t *testing.T) {
	originalResolve := resolveFunc
	defer func() {
		resolveFunc = originalResolve
	}()

	resolveFunc = func(id string) (record, error) {
		return record{
			Bundle:    id,
			Name:      "My\nApp",
			Publisher: "Dev",
			URL:       "https://example.com/app",
		}, nil
	}

	input := strings.NewReader("999\n")
	var out strings.Builder
	fields := []Field{FieldBundle, FieldName, FieldPublisher, FieldURL}

	if err := process(input, &out, fields, true, false, false); err != nil {
		t.Fatalf("process returned error: %v", err)
	}

	got := out.String()
	want := "bundle\tname\tpublisher\turl\n999\tMy App\tDev\thttps://example.com/app\n"
	if got != want {
		t.Fatalf("tsv output mismatch:\n got: %q\nwant: %q", got, want)
	}
}
