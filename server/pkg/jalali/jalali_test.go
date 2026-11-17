package jalali

import (
	"testing"
	"time"
)

func TestKnownDates(t *testing.T) {
	for _, tc := range []struct {
		gregorian string
		want      Date
	}{
		{"2025-03-21", Date{1404, 1, 1}},
		{"2026-03-21", Date{1405, 1, 1}},
		{"2026-11-16", Date{1405, 8, 25}},
		{"2025-03-20", Date{1403, 12, 30}}, // 1403 is a leap year
		{"2000-01-01", Date{1378, 10, 11}},
	} {
		day, _ := time.Parse(time.DateOnly, tc.gregorian)
		if got := FromGregorian(day); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.gregorian, got, tc.want)
		}
		if got := ToGregorian(tc.want.Year, tc.want.Month, tc.want.Day); !got.Equal(day) {
			t.Errorf("%v: got %s, want %s", tc.want, got.Format(time.DateOnly), tc.gregorian)
		}
	}
}

// Every day converts to Jalali and back, and Jalali days follow each other.
func TestRoundTrip(t *testing.T) {
	day := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	previous := FromGregorian(day.AddDate(0, 0, -1))
	for range 365 * 70 {
		date := FromGregorian(day)
		if back := ToGregorian(date.Year, date.Month, date.Day); !back.Equal(day) {
			t.Fatalf("%s -> %v -> %s", day.Format(time.DateOnly), date, back.Format(time.DateOnly))
		}
		next := date.Day == previous.Day+1 && date.Month == previous.Month && date.Year == previous.Year
		newMonth := date.Day == 1 && (date.Month == previous.Month+1 && date.Year == previous.Year ||
			date.Month == 1 && previous.Month == 12 && date.Year == previous.Year+1)
		if !next && !newMonth {
			t.Fatalf("%v does not follow %v", date, previous)
		}
		previous = date
		day = day.AddDate(0, 0, 1)
	}
}

func TestMonthBounds(t *testing.T) {
	day := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC) // 1405/08/25
	if got := MonthStart(day).Format(time.DateOnly); got != "2026-10-23" {
		t.Fatalf("month start %s", got)
	}
	if got := NextMonthStart(day).Format(time.DateOnly); got != "2026-11-22" {
		t.Fatalf("next month start %s", got)
	}
	esfand := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) // 1404/12/10
	if got := NextMonthStart(esfand).Format(time.DateOnly); got != "2026-03-21" {
		t.Fatalf("next month of Esfand %s", got)
	}
}
