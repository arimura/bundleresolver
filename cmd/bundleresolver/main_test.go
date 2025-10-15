package main

import "testing"

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
