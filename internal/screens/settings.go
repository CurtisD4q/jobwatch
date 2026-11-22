package screens

import (
	"time"

	mint "github.com/CurtisD4q/mint-kit"
	"github.com/CurtisD4q/mint-loom/health"
)

const (
	muteDay   = "Mute it for a day"
	muteWeek  = "Mute it for a week"
	muteFor   = "Mute it until I lift it"
	muteLift  = "Lift the mute"
	muteChose = "A muted finding still shows here. It is not sent, by the background check or by a notification."
)

// settingsRows mute or unmute each finding that would be sent about the
// job, and delete its history.
func settingsRows(o Options, name string, fs []health.Finding, mutes []health.Mute) []mint.Row {
	var rows []mint.Row
	seen := map[string]bool{}
	for _, f := range fs {
		if f.Job != name || f.Level < health.Warn || seen[f.ID()] {
			continue
		}
		seen[f.ID()] = true
		f := f
		state := "sent"
		if ok, until := health.Muted(mutes, f); ok {
			state = "muted"
			if !until.IsZero() {
				state = "muted to " + until.Local().Format("2 Jan 15:04")
			}
		}
		label := "Alerts about " + string(f.Kind)
		if f.Other != "" {
			label += " (" + f.Other + ")"
		}
		rows = append(rows, mint.Action(mint.ActionOpts{ID: "mute-" + f.ID(), Label: label,
			Get: func() string { return state },
			OnActivate: func(ctx *mint.Ctx) {
				pick, ok := ctx.Choose(string(f.Kind)+": "+f.Says, []string{muteDay, muteWeek, muteFor, muteLift}, muteChose)
				if !ok {
					return
				}
				now := o.now()
				var err error
				switch pick {
				case muteDay:
					err = health.SetMute(o.Book.Dir, health.MuteOf(f, now.Add(24*time.Hour)), now)
				case muteWeek:
					err = health.SetMute(o.Book.Dir, health.MuteOf(f, now.Add(7*24*time.Hour)), now)
				case muteFor:
					err = health.SetMute(o.Book.Dir, health.MuteOf(f, time.Time{}), now)
				case muteLift:
					err = health.Unmute(o.Book.Dir, health.MuteOf(f, time.Time{}), now)
				}
				if err != nil {
					ctx.ToastError(err.Error())
					return
				}
				ctx.Toast("Saved.")
			}}))
	}
	rows = append(rows, mint.Action(mint.ActionOpts{ID: "forget", Label: "Delete its history",
		Get: func() string { return "cannot be undone" },
		OnActivate: func(ctx *mint.Ctx) {
			a, err := o.Book.ForgetAction(name)
			if err != nil {
				ctx.ToastError(err.Error())
				return
			}
			if out := o.do(ctx, a); out.OK() {
				ctx.Pop()
			}
		}}))
	return rows
}
