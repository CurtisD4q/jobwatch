package main

import (
	"reflect"
	"testing"
)

func TestOpenScreen(t *testing.T) {
	cases := []struct{ in, want []string }{
		// No arguments opens the job list rather than the menu.
		{[]string{"jobwatch"}, []string{"jobwatch", "--screen", "Overview"}},
		// --dir is ours: it does not count as asking for a screen.
		{[]string{"jobwatch", "--dir", "/tmp/x"}, []string{"jobwatch", "--screen", "Overview"}},
		{[]string{"jobwatch", "--dir=/tmp/x"}, []string{"jobwatch", "--screen", "Overview"}},
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

// A command runs and exits; anything else opens the screens, so Mint's own
// flags and a screen name are never taken for a command.
func TestIsCommand(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"record", "nightly", "--", "true"}, true},
		{[]string{"check", "--shadow"}, true},
		{[]string{"import", "journal", "--unit", "backup.service"}, true},
		{[]string{"--screen", "Jobs"}, false},
		{[]string{"--version"}, false},
		{[]string{"--help"}, false},
		{[]string{"Jobs"}, false},
		{[]string{"help"}, true},
	} {
		if got := isCommand(c.args); got != c.want {
			t.Errorf("isCommand(%q) = %v, want %v", c.args, got, c.want)
		}
	}
	// --dir comes off before the command is looked for
	if !isCommand(flagRest([]string{"--dir", "/tmp/x", "report"})) {
		t.Error("--dir in front of a command hid it")
	}
}

// Nothing after "--" is jobwatch's: it is the recorded command's own.
func TestDirStopsAtDashDash(t *testing.T) {
	args := []string{"record", "sync", "--", "rsync", "--dir", "/x", "-a"}
	if d := flagDir(args); d != "" {
		t.Fatalf("took %q from the command", d)
	}
	if got := flagRest(args); !reflect.DeepEqual(got, args) {
		t.Fatalf("%q", got)
	}
	if d := flagDir(append([]string{"--dir", "/data"}, args...)); d != "/data" {
		t.Fatal(d)
	}
}
