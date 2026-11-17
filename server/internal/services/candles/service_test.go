package candles

import (
	"testing"
	"time"
)

// TestWeekStart checks that TSE weeks start on Saturday, like the weekly
// candles built in ClickHouse.
func TestWeekStart(t *testing.T) {
	saturday := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	for offset := range 7 {
		day := saturday.AddDate(0, 0, offset)
		if got := weekStart(day); !got.Equal(saturday) {
			t.Fatalf("week of %s (%s) starts %s, want %s", day.Format(dateLayout), day.Weekday(), got.Format(dateLayout), saturday.Format(dateLayout))
		}
	}
	if got := weekStart(saturday.AddDate(0, 0, 7)); !got.Equal(saturday.AddDate(0, 0, 7)) {
		t.Fatalf("next Saturday starts week %s", got.Format(dateLayout))
	}
}

// TestMonthBuilds checks that monthly builds cover whole Jalali months, each
// with the starts of its months and of the month after it.
func TestMonthBuilds(t *testing.T) {
	first := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)  // 1405/04/10
	last := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC) // 1405/08/19
	builds := monthBuilds(first, last, 2, nil)
	want := [][2]string{{"2026-06-22", "2026-08-22"}, {"2026-08-23", "2026-10-22"}, {"2026-10-23", "2026-11-21"}}
	if len(builds) != len(want) {
		t.Fatalf("got %d builds, want %d", len(builds), len(want))
	}
	for i, build := range builds {
		if got := [2]string{build.From.Format(dateLayout), build.To.Format(dateLayout)}; got != want[i] {
			t.Fatalf("build %d covers %v, want %v", i, got, want[i])
		}
		starts := build.MonthStarts
		if !starts[0].Equal(build.From) || !starts[len(starts)-1].Equal(build.To.AddDate(0, 0, 1)) {
			t.Fatalf("build %d month starts %v", i, starts)
		}
	}
}
