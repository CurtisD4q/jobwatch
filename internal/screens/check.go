package screens

import (
	"context"
	"strings"

	"github.com/CurtisD4q/jobwatch/internal/jw"
	mint "github.com/CurtisD4q/pmin-kit"
	loom "github.com/CurtisD4q/pmin-loom"
)

func (o Options) jwOpts() jw.Options {
	return jw.Options{Dir: o.Book.Dir, Exec: o.Book.Exec, Program: o.Program, Home: o.Home, Now: o.Book.Now}
}

// BackgroundCheck turns the check that sends alerts on and off, and sets
// whether it only shadows and whether it notifies. Turning it on or off is
// an action with its change shown first and an undo; the two modes are
// saved settings that the next run picks up.
func BackgroundCheck(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "⏲", Title: "Background check"}
	scr.Refresh = func(s *mint.Screen) {
		cs := o.checkSettings()
		state := "off"
		if cs.On {
			state = "on, " + cs.Schedule + " (" + cs.How + ")"
		} else if cs.Saved {
			state = "off (was " + cs.Schedule + ")"
		}
		mode := "sends alerts"
		if cs.Shadow {
			mode = "shadow, sends nothing"
		}
		notify := "off"
		if cs.Notify {
			notify = "on"
		}
		rows := []mint.Row{
			mint.Action(mint.ActionOpts{ID: "on", Label: "Check", Get: func() string { return state },
				OnActivate: func(ctx *mint.Ctx) {
					if cs.On {
						a, err := jw.UnscheduleCheck(o.jwOpts())
						if err != nil {
							ctx.ToastError(err.Error())
							return
						}
						o.do(ctx, a)
						return
					}
					if why := o.cannotChange(); why != "" {
						ctx.Message("Cannot schedule from here", capital(why)+".")
						return
					}
					def := cs.Schedule
					if def == "" {
						def = "hourly"
						if !loom.HasUserSystemd(context.Background(), o.Book.Exec) {
							def = "0 * * * *"
						}
					}
					spec, ok := ctx.Prompt("Run the check", "When: a systemd time such as hourly, or cron fields such as 0 * * * *", def)
					spec = strings.TrimSpace(spec)
					if !ok || spec == "" {
						return
					}
					a, err := jw.ScheduleCheck(o.jwOpts(), spec)
					if err != nil {
						ctx.ToastError(err.Error())
						return
					}
					o.do(ctx, a)
				}}),
			mint.Action(mint.ActionOpts{ID: "mode", Label: "Mode", Get: func() string { return mode },
				OnActivate: func(ctx *mint.Ctx) { setModes(ctx, o, !cs.Shadow, cs.Notify) }}),
			mint.Action(mint.ActionOpts{ID: "notify", Label: "Desktop notifications", Get: func() string { return notify },
				OnActivate: func(ctx *mint.Ctx) { setModes(ctx, o, cs.Shadow, !cs.Notify) }}),
		}
		var about []mint.Row
		about = append(about, wrapped("Shadow mode writes what the check would send, and sends nothing. Use it to try jobwatch on real jobs: compare what it would have sent with what happened, then switch to sending.", "")...)
		about = append(about, wrapped("Alerts are written to "+jw.AlertsPath(o.jwOpts())+" with the time, and printed, so cron mails them and a timer puts them in the journal.", "")...)
		about = append(about,
			mint.Action(mint.ActionOpts{ID: "alerts", Label: "Alerts", Get: func() string { return "sent and shadow" },
				OnActivate: func(ctx *mint.Ctx) { ctx.Push(Alerts(o)) }}))
		s.Sections = []mint.Section{{Rows: rows}, {Rows: about}}
	}
	return o.withKeys(scr)
}

func setModes(ctx *mint.Ctx, o Options, shadow, notify bool) {
	defer o.changed()
	if err := jw.SetCheckModes(o.jwOpts(), shadow, notify); err != nil {
		ctx.ToastError(err.Error() + ": turn the check on first")
		return
	}
	ctx.Toast("Saved. The next check uses it.")
}

// Alerts is what the check sent, and what a shadow check would have sent,
// newest first.
func Alerts(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "!", Title: "Alerts"}
	scr.Poll = o.live()
	scr.Refresh = func(s *mint.Screen) {
		all := jw.RecentAlerts(o.jwOpts(), 50)
		var sent, shadow []mint.Row
		for _, a := range all {
			rows := wrapped(a.When.Local().Format("Mon 2 Jan 15:04")+"  "+a.Text, "")
			if a.Shadow {
				shadow = append(shadow, rows...)
			} else {
				sent = append(sent, rows...)
			}
		}
		if len(sent) == 0 {
			sent = []mint.Row{mint.Note("Nothing sent yet.")}
		}
		if len(shadow) == 0 {
			shadow = []mint.Row{mint.Note("Nothing from shadow checks.")}
		}
		s.Sections = []mint.Section{{Name: "Sent", Rows: sent}, {Name: "Shadow: not sent", Rows: shadow}}
	}
	return o.withKeys(scr)
}
