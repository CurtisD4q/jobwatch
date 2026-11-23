package screens

import (
	"time"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
	mint "github.com/CurtisD4q/pmin-kit"
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
func settingsRows(o Options, name string, fs []jobhealth.Finding, mutes []jobhealth.Mute) []mint.Row {
	var rows []mint.Row
	seen := map[string]bool{}
	for _, f := range fs {
		if f.Job != name || f.Level < jobhealth.Warn || seen[f.ID()] {
			continue
		}
		seen[f.ID()] = true
		f := f
		state := "sent"
		if ok, until := jobhealth.Muted(mutes, f); ok {
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
					err = jobhealth.SetMute(o.Book.Dir, jobhealth.MuteOf(f, now.Add(24*time.Hour)), now)
				case muteWeek:
					err = jobhealth.SetMute(o.Book.Dir, jobhealth.MuteOf(f, now.Add(7*24*time.Hour)), now)
				case muteFor:
					err = jobhealth.SetMute(o.Book.Dir, jobhealth.MuteOf(f, time.Time{}), now)
				case muteLift:
					err = jobhealth.Unmute(o.Book.Dir, jobhealth.MuteOf(f, time.Time{}), now)
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
