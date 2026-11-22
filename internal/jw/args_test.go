package jw

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A recorded command keeps every argument it was given, including ones
// that look like jobwatch's own flags. Before, --dir was taken out of it,
// and the run went to a history in a folder of that name.
func TestRecordedCommandKeepsItsArguments(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "argv")
	script := filepath.Join(t.TempDir(), "show")
	os.WriteFile(script, []byte("#!/bin/sh\necho \"$@\" > "+out+"\n"), 0o700)
	o := opts(dir)
	if code := Main([]string{"--dir", dir, "record", "args", "--", script, "one", "--dir", "two", "--quiet", "--name", "x"}, o); code != 0 {
		_, e := buffers(o)
		t.Fatalf("exit %d: %s", code, e)
	}
	got, _ := os.ReadFile(out)
	if strings.TrimSpace(string(got)) != "one --dir two --quiet --name x" {
		t.Fatalf("the command got %q", got)
	}
	if _, err := os.Stat("two"); err == nil {
		os.RemoveAll("two")
		t.Fatal("a history was written to ./two")
	}
	if runs, _ := opts(dir).book().History("args", 0); len(runs) != 1 {
		t.Fatalf("%d runs in the history given with --dir", len(runs))
	}
}

func TestSplitRecord(t *testing.T) {
	for _, c := range []struct {
		in    []string
		name  string
		quiet bool
		cmd   []string
		ok    bool
	}{
		{[]string{"nightly", "--", "true"}, "nightly", false, []string{"true"}, true},
		{[]string{"--quiet", "nightly", "--", "true", "--quiet"}, "nightly", true, []string{"true", "--quiet"}, true},
		{[]string{"nightly", "--quiet", "--", "true"}, "nightly", true, []string{"true"}, true},
		{[]string{"--name", "nightly", "--", "true"}, "nightly", false, []string{"true"}, true},
		{[]string{"--name=nightly", "--", "true"}, "nightly", false, []string{"true"}, true},
		{[]string{"nightly", "true"}, "", false, nil, false},                // no --
		{[]string{"nightly", "--"}, "", false, nil, false},                  // nothing to run
		{[]string{"--name", "a", "b", "--", "true"}, "", false, nil, false}, // two names
		{[]string{"--bogus", "a", "--", "true"}, "", false, nil, false},
	} {
		name, quiet, cmd, ok := splitRecord(c.in)
		if ok != c.ok || name != c.name || quiet != c.quiet || !reflect.DeepEqual(cmd, c.cmd) {
			t.Errorf("splitRecord(%q) = %q %v %q %v", c.in, name, quiet, cmd, ok)
		}
	}
}

// Flags may come before or after the words, and an unknown flag is a
// usage error rather than ignored.
func TestFlagsInAnyOrder(t *testing.T) {
	o := opts(t.TempDir())
	if code := Main([]string{"report", "--json"}, o); code != 0 {
		t.Fatalf("report --json: %d", code)
	}
	for _, args := range [][]string{
		{"report", "--bogus"},
		{"report", "extra"},
		{"check", "--file"},
		{"respace", "--window", "soon"},
		{"respace", "--window", "-5m"},
		{"import", "journal"},
		{"import", "rsync", "file.log"},
		{"schedule", "--on", "--undo", "hourly"},
	} {
		if code := Main(args, opts(t.TempDir())); code != 2 {
			t.Errorf("%q: exit %d, want 2", args, code)
		}
	}
	// a word after the flags, or before them
	for _, args := range [][]string{{"schedule", "hourly", "--json"}, {"schedule", "--json", "hourly"}} {
		o := opts(t.TempDir())
		code := Main(args, o)
		_, e := buffers(o)
		if code == 2 {
			t.Errorf("%q read as a usage error: %s", args, e)
		}
	}
}
