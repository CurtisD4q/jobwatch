// Package screens is jobwatch's interface: the job list, a job's evidence,
// runs and settings, the day, re-spacing, adding a job from the crontab,
// the background check and its alerts. It reads mint-loom's health checks
// and records; it never changes a schedule on its own.
package screens

import (
	"time"

	"github.com/CurtisD4q/jobwatch/internal/jw"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
	loom "github.com/CurtisD4q/pmin-loom"
)

// Options configure the screens.
type Options struct {
	Book *loom.Book
	// Program is jobwatch's absolute path, which a schedule calls. Empty
	// turns scheduling and crontab changes off.
	Program string
	// Home is the history cron lines record into (see jw.Options.Home).
	Home string

	// Version is jobwatch's version, shown on Keys and settings.
	Version string

	snap *snapshot // shared by the screens Entries makes; nil: each works alone
}

// cannotChange is why jobwatch will not change the crontab or schedule
// itself from here, or "" when it can.
func (o Options) cannotChange() string {
	if o.Program == "" {
		return "this program does not know its own path, so it cannot change your crontab"
	}
	return jw.Elsewhere(o.Book.Dir, o.Home)
}

// command is what a person types to record a job.
const command = "jobwatch"

func (o Options) command() string { return command }

func (o Options) now() time.Time {
	if o.Book.Now != nil {
		return o.Book.Now()
	}
	return time.Now()
}

// describe is a job's state and the finding behind it, in a line.
func describe(st jobhealth.Status) string {
	s := st.State
	if st.Says != "" {
		s += ". " + st.Says
	}
	return s
}

func (o Options) version() string {
	if o.Version == "" {
		return "unknown"
	}
	return o.Version
}

// capital is s with its first letter a capital, to start a sentence.
func capital(s string) string {
	r := []rune(s)
	if len(r) > 0 && r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}
