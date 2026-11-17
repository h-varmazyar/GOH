// Package jalali converts between Gregorian dates and the Jalali (Solar
// Hijri) calendar TSE trades on, with the arithmetic 33-year cycle.
package jalali

import "time"

// Date is a Jalali date.
type Date struct {
	Year, Month, Day int
}

// ToGregorian returns the Gregorian date of a Jalali date at UTC midnight.
func ToGregorian(jy, jm, jd int) time.Time {
	jy += 1595
	days := -355668 + 365*jy + (jy/33)*8 + ((jy%33)+3)/4 + jd
	if jm < 7 {
		days += (jm - 1) * 31
	} else {
		days += (jm-7)*30 + 186
	}
	gy := 400 * (days / 146097)
	days %= 146097
	if days > 36524 {
		days--
		gy += 100 * (days / 36524)
		days %= 36524
		if days >= 365 {
			days++
		}
	}
	gy += 4 * (days / 1461)
	days %= 1461
	if days > 365 {
		gy += (days - 1) / 365
		days = (days - 1) % 365
	}
	// days is the zero-based day of the Gregorian year; time.Date normalizes the overflow.
	return time.Date(gy, time.January, days+1, 0, 0, 0, 0, time.UTC)
}

var gregorianMonthDays = [12]int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}

// FromGregorian returns the Jalali date of the calendar day of t.
func FromGregorian(t time.Time) Date {
	gy, gm, gd := t.Year(), int(t.Month()), t.Day()
	gy2 := gy
	if gm > 2 {
		gy2++
	}
	days := 355666 + 365*gy + (gy2+3)/4 - (gy2+99)/100 + (gy2+399)/400 + gd + gregorianMonthDays[gm-1]
	jy := -1595 + 33*(days/12053)
	days %= 12053
	jy += 4 * (days / 1461)
	days %= 1461
	if days > 365 {
		jy += (days - 1) / 365
		days = (days - 1) % 365
	}
	if days < 186 {
		return Date{Year: jy, Month: 1 + days/31, Day: 1 + days%31}
	}
	return Date{Year: jy, Month: 7 + (days-186)/30, Day: 1 + (days-186)%30}
}

// MonthStart returns the Gregorian date, at UTC midnight, of the first day of
// the Jalali month of the calendar day of t.
func MonthStart(t time.Time) time.Time {
	date := FromGregorian(t)
	return ToGregorian(date.Year, date.Month, 1)
}

// NextMonthStart returns the first day of the Jalali month after the month of t.
func NextMonthStart(t time.Time) time.Time {
	date := FromGregorian(t)
	if date.Month == 12 {
		return ToGregorian(date.Year+1, 1, 1)
	}
	return ToGregorian(date.Year, date.Month+1, 1)
}
