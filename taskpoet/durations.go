package taskpoet

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// parseDuration parses a duration string in a way very similar to TaskWarrior,
// documented here: https://taskwarrior.org/docs/durations/
func parseDuration(s string) (*time.Duration, error) {
	if s == "" {
		return nil, errors.New("duration must not be an empty string")
	}
	r := regexp.MustCompile(`(?P<ordinal>\d+)?\s?(?P<unit>\w+)`)
	matches := r.FindStringSubmatch(s)
	// fmt.Fprintf(os.Stderr, "MATCHES: %+v\n", matches)
	ordinal := 1
	if matches[1] != "" {
		var err error
		if ordinal, err = strconv.Atoi(matches[1]); err != nil {
			return nil, err
		}
	}
	unit := matches[2]
	switch unit {
	case "seconds", "second", "secs", "sec", "s":
		d := time.Duration(ordinal) * time.Second
		return &d, nil
	case "minutes", "minute", "mins", "min":
		d := time.Duration(ordinal) * time.Minute
		return &d, nil
	case "hours", "hour", "hrs", "hr", "h":
		d := time.Duration(ordinal) * time.Hour
		return &d, nil
	case "days", "day", "d", "daily":
		d := time.Duration(ordinal) * (24 * time.Hour)
		return &d, nil
	case "weeks", "week", "wks", "wk", "w":
		d := time.Duration(ordinal) * ((24 * time.Hour) * 7)
		return &d, nil
	case "monthly", "months", "month", "mnths", "mths", "mth", "mo", "m":
		d := time.Duration(ordinal) * ((24 * time.Hour) * 30)
		return &d, nil
	case "quarterly", "quarters", "quarter", "qrtrs", "qrtr", "qtr", "q":
		d := time.Duration(ordinal) * ((24 * time.Hour) * 91)
		return &d, nil
	case "semiannual":
		d := time.Duration(ordinal) * ((24 * time.Hour) * 180)
		return &d, nil
	case "yearly", "years", "year", "yrs", "yr", "y":
		d := time.Duration(ordinal) * (time.Hour * 8760)
		return &d, nil
	default:
		return nil, fmt.Errorf("invalid unit: %v", unit)
	}
}
