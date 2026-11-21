package main

import (
	"reflect"
	"testing"
)

func TestOpenScreen(t *testing.T) {
	cases := []struct{ in, want []string }{
		// No arguments opens the job list rather than the menu.
		{[]string{"jobwatch"}, []string{"jobwatch", "--screen", "Jobs"}},
		// --dir is ours: it does not count as asking for a screen.
		{[]string{"jobwatch", "--dir", "/tmp/x"}, []string{"jobwatch", "--screen", "Jobs"}},
		{[]string{"jobwatch", "--dir=/tmp/x"}, []string{"jobwatch", "--screen", "Jobs"}},
		// A screen asked for is left alone, in either form, wherever it is.
		{[]string{"jobwatch", "--screen", "The day"}, []string{"jobwatch", "--screen", "The day"}},
		{[]string{"jobwatch", "--dir", "/tmp/x", "--screen", "The day"},
			[]string{"jobwatch", "--screen", "The day"}},
		{[]string{"jobwatch", "--screen=The day"}, []string{"jobwatch", "--screen=The day"}},
		// Mint's own flags are not ours to rewrite.
		{[]string{"jobwatch", "--help"}, []string{"jobwatch", "--help"}},
		{[]string{"jobwatch", "--version"}, []string{"jobwatch", "--version"}},
	}
	for _, c := range cases {
		if got := openScreen(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("openScreen(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A --dir with nothing after it is dropped rather than read as the next
// argument, and a screen still opens.
func TestOpenScreenTrailingDir(t *testing.T) {
	if got := openScreen([]string{"jobwatch", "--screen", "Jobs", "--dir"}); !reflect.DeepEqual(got,
		[]string{"jobwatch", "--screen", "Jobs"}) {
		t.Fatalf("%q", got)
	}
}

// The dir is read from either form, and "" when it was not given.
func TestFlagDir(t *testing.T) {
	cases := map[string]struct {
		in   []string
		want string
	}{
		"none":     {[]string{"jobwatch"}, ""},
		"separate": {[]string{"jobwatch", "--dir", "/tmp/x"}, "/tmp/x"},
		"joined":   {[]string{"jobwatch", "--dir=/tmp/x"}, "/tmp/x"},
		"empty":    {[]string{"jobwatch", "--dir"}, ""},
		"later":    {[]string{"jobwatch", "--screen", "Jobs", "--dir", "/tmp/y"}, "/tmp/y"},
	}
	for name, c := range cases {
		if got := flagDir(c.in); got != c.want {
			t.Errorf("%s: flagDir(%q) = %q, want %q", name, c.in, got, c.want)
		}
	}
}
