// Package screens is jobwatch's interface: the job list, a job's evidence,
// runs and settings, the day, re-spacing, adding a job from the crontab,
// the background check and its alerts. It reads mint-loom's health checks
// and records; it never changes a schedule on its own.
package screens

import (
	"time"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/health"
)

// Options configure the screens.
type Options struct {
	Book *loom.Book
	// Program is jobwatch's absolute path, which a schedule calls. Empty
	// turns scheduling and crontab changes off.
	Program string

	// Version is jobwatch's version, shown on Keys and settings.
	Version string

	snap *snapshot // shared by the screens Entries makes; nil: each works alone
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
func describe(st health.Status) string {
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
