package tse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseJalaliDate parses a YYYY/MM/DD Jalali date into a Gregorian date at UTC midnight.
func parseJalaliDate(value string) (time.Time, error) {
	parts := strings.Split(strings.TrimSpace(persianReplacer.Replace(value)), "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("expected YYYY/MM/DD, got %q", value)
	}
	var values [3]int
	for i, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid Jalali date %q", value)
		}
		values[i] = parsed
	}
	jy, jm, jd := values[0], values[1], values[2]
	if jy < 1 || jm < 1 || jm > 12 || jd < 1 || jd > 31 || (jm > 6 && jd > 30) {
		return time.Time{}, fmt.Errorf("invalid Jalali date %q", value)
	}
	return jalaliToGregorian(jy, jm, jd), nil
}

func jalaliToGregorian(jy, jm, jd int) time.Time {
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
