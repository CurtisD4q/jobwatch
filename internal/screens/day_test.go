package screens

import (
	"strings"
	"testing"

	"github.com/CurtisD4q/mint-loom/health"
)

// Where the next job starts is one column, even when the start is read
// back a hair under the hour.
func TestNextJobIsOneMark(t *testing.T) {
	b := health.Bar{Name: "sync", Period: 1440, Start: 180, Usual: 206, Slow: 209, Next: 239.99999999999994, NextName: "prune", Gap: 31}
	if got := strings.Count(drawBar(b, 66), "┆"); got != 1 {
		t.Fatalf("%d marks: %q", got, drawBar(b, 66))
	}
}
