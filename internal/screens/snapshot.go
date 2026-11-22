package screens

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/CurtisD4q/jobwatch/internal/jw"
	mint "github.com/CurtisD4q/mint-kit"
	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/health"
	"github.com/CurtisD4q/mint-tools/action"
	toolsui "github.com/CurtisD4q/mint-tools/ui"
)

// A screen is redrawn on every key, and its Refresh runs each time. Reading
// every history, running the whole health check and asking crontab on each
// key was wasteful, and each screen doing its own check meant two views
// could, in a race, show two answers. So the screens share one snapshot:
// one check of one history, worked out again only when the history, the
// recipes or the mutes change on disk, when the minute changes (findings
// such as "overdue" depend on the time), or when a screen changes
// something itself and says so.

// cronTTL is how long the crontab read is reused. A change made through
// jobwatch invalidates it at once; one made by hand shows within this.
const cronTTL = 30 * time.Second

type snapshot struct {
	mu    sync.Mutex
	stamp string
	jobs  []health.Job
	rep   health.Report
	mutes []health.Mute
	err   error

	cronAt  time.Time
	cron    []loom.CronLine
	cronErr error

	settingsAt time.Time
	settings   jw.CheckSettings
}

func newSnapshot() *snapshot { return &snapshot{} }

// stampOf is what the check depends on: the files under the history, the
// recipes and the health state, by name, size and time, and the minute.
func stampOf(dir string, now time.Time) string {
	var b strings.Builder
	for _, sub := range []string{"runs", filepath.Join("runs", "imported"), "recipes", "health"} {
		entries, _ := os.ReadDir(filepath.Join(dir, sub))
		for _, e := range entries {
			info, err := e.Info()
			if err != nil || info.IsDir() {
				continue
			}
			fmt.Fprintf(&b, "%s/%s %d %d\n", sub, e.Name(), info.Size(), info.ModTime().UnixNano())
		}
	}
	fmt.Fprintf(&b, "%d", now.Truncate(time.Minute).Unix())
	return b.String()
}

// check is the shared result: the jobs, the health report and the mutes.
func (o Options) check() ([]health.Job, health.Report, []health.Mute, error) {
	s := o.snap
	if s == nil {
		s = newSnapshot() // a screen made on its own: no sharing, same answer
	}
	now := o.now()
	stamp := stampOf(o.Book.Dir, now)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stamp == stamp && s.stamp != "" {
		return s.jobs, s.rep, s.mutes, s.err
	}
	jobs, err := health.FromBook(context.Background(), o.Book, now)
	s.jobs, s.err, s.stamp = jobs, err, stamp
	s.rep, s.mutes = health.Report{}, nil
	if err == nil {
		s.rep = health.Check(jobs, now)
		s.mutes = health.Mutes(o.Book.Dir, now)
	}
	return s.jobs, s.rep, s.mutes, s.err
}

// cronLines is the person's crontab, read at most once every cronTTL.
func (o Options) cronLines() ([]loom.CronLine, error) {
	s := o.snap
	if s == nil {
		return loom.CronLines(context.Background(), o.Book.Exec)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.cronAt.IsZero() && time.Since(s.cronAt) < cronTTL {
		return s.cron, s.cronErr
	}
	s.cron, s.cronErr = loom.CronLines(context.Background(), o.Book.Exec)
	s.cronAt = time.Now()
	return s.cron, s.cronErr
}

// checkSettings is the background check's settings, which read the
// crontab or the timer, reused for cronTTL like the crontab itself.
func (o Options) checkSettings() jw.CheckSettings {
	s := o.snap
	if s == nil {
		return jw.ReadCheck(o.jwOpts())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.settingsAt.IsZero() && time.Since(s.settingsAt) < cronTTL {
		return s.settings
	}
	s.settings, s.settingsAt = jw.ReadCheck(o.jwOpts()), time.Now()
	return s.settings
}

// changed says the screens' view of the history or the crontab is out of
// date, after jobwatch itself changed one of them.
func (o Options) changed() {
	if s := o.snap; s != nil {
		s.mu.Lock()
		s.stamp, s.cronAt, s.settingsAt = "", time.Time{}, time.Time{}
		s.mu.Unlock()
	}
}

// do runs an action with its preview and confirmation, then marks the
// shared view out of date, since the action may have changed the crontab
// or the history.
func (o Options) do(ctx *mint.Ctx, a action.Action) action.Outcome {
	out := toolsui.Do(ctx, action.Doer{Exec: o.Book.Exec, Record: o.Book.Record}, a)
	o.changed()
	return out
}
