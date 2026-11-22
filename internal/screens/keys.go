package screens

import (
	"os"
	"path/filepath"

	mint "github.com/CurtisD4q/mint-kit"
)

// jump is a key that opens a screen, the same on every jobwatch screen.
type jump struct {
	key, what string
	open      func(Options) *mint.Screen
}

// jumps are jobwatch's keys. None is one Mint keeps for itself (b, q, y,
// space, H, L, [, ], |) or the vim style uses (j, k, h, l, g, G). It is a
// function, not a variable, because the screens it opens use it too.
func jumps() []jump {
	return []jump{
		{"o", "Overview", Overview},
		{"d", "The day", Day},
		{"a", "Alerts", Alerts},
		{"c", "Background check", BackgroundCheck},
		{"n", "Add a job", AddJobs},
		{"i", "Import history", Import},
		{"?", "How this works", Guide},
		{"s", "Keys and settings", Settings},
	}
}

const keyHints = " ↑↓ move   ⏎ open   d day   a alerts   ? help   b back   q quit "

// withKeys gives a screen jobwatch's keys: each opens its screen (not the
// one already showing), and r reads everything again.
func (o Options) withKeys(scr *mint.Screen) *mint.Screen {
	for _, j := range jumps() {
		j := j
		if j.what == scr.Title {
			continue
		}
		scr.Keys = append(scr.Keys, mint.ScreenKey{Key: j.key, Do: func(ctx *mint.Ctx) { ctx.Push(j.open(o)) }})
	}
	scr.Keys = append(scr.Keys, mint.ScreenKey{Key: "r", Do: func(ctx *mint.Ctx) {
		o.changed()
		ctx.Toast("Read again.")
	}})
	if scr.Hints == "" {
		scr.Hints = keyHints
	}
	return scr
}

// Settings is the keys, where jobwatch keeps things, and how it looks.
func Settings(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "⚙", Title: "Keys and settings"}
	scr.Refresh = func(s *mint.Screen) {
		var keys []mint.Row
		for _, j := range jumps() {
			keys = append(keys, mint.Note(j.key+"   "+j.what))
		}
		keys = append(keys,
			mint.Note("r   read the history and crontab again"),
		)
		keys = append(keys, wrapped("↑↓ move, ⏎ open or change, Tab next pane, [ ] pane size, y copy, b back, q quit", "")...)
		style := os.Getenv("MINT_KEYS")
		if style == "" {
			style = "default"
		}
		dir := o.Book.Dir
		kept := []mint.Row{
			mint.Value(mint.ValueOpts{Label: "History", Get: func() string { return dir }}),
			mint.Value(mint.ValueOpts{Label: "Alerts sent", Get: func() string { return filepath.Join(dir, "health", "alerts.log") }}),
			mint.Value(mint.ValueOpts{Label: "Shadow alerts", Get: func() string { return filepath.Join(dir, "health", "shadow.log") }}),
			mint.Value(mint.ValueOpts{Label: "Mutes", Get: func() string { return filepath.Join(dir, "health", "mute.json") }}),
		}
		kept = append(kept, wrapped("Another folder: start jobwatch with --dir PATH. Files are private to you.", "")...)
		look := []mint.Row{
			mint.Value(mint.ValueOpts{Label: "Key style", Get: func() string { return style }}),
		}
		look = append(look, wrapped("Set MINT_KEYS=emacs or vim for those key styles. Colours are jobwatch's own until you choose a theme on Theme, or set MINT_PALETTE.", "")...)
		about := []mint.Row{mint.Value(mint.ValueOpts{Label: "Version", Get: func() string { return o.version() }})}
		s.Sections = []mint.Section{{Name: "Keys", Rows: keys}, {Name: "Where things are kept", Rows: kept}, {Name: "Look", Rows: look}, {Name: "jobwatch", Rows: about}}
	}
	return o.withKeys(scr)
}
