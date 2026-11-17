package tse

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/pkg/jalali"
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

func jalaliToGregorian(jy, jm, jd int) time.Time { return jalali.ToGregorian(jy, jm, jd) }
