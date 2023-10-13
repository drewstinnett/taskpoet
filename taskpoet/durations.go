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

// Synonym is a shorthand expression for a specific datetime
// type Synonym func() *time.Time

// Calendar is a little helper function that can work with synonym times
type Calendar struct {
	present time.Time
}

// WithPresent sets the present time to an arbitrary datetime
func WithPresent(t *time.Time) func(*Calendar) {
	return func(c *Calendar) {
		c.present = *t
	}
}

// Synonym returns a time.Time from a string
// Reference: https://taskwarrior.org/docs/dates/
func (c Calendar) Synonym(s string) (time.Time, error) {
	syn, err := c.getSynonymer(s)
	if err == nil {
		return syn(&c.present), nil
	}
	return time.Time{}, fmt.Errorf("unknown synonym: %v", s)
}

func datePTR(t time.Time) *time.Time {
	return &t
}

// Date returns a date in the future or past based on the given string. Can be a
// Synonym or any valid go time.Duration string
func (c Calendar) Date(s string) (*time.Time, error) {
	syn, err := c.Synonym(s)
	if err != nil {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, err
		}
		return datePTR(c.present.Add(d)), nil
	}
	return &syn, nil
}

func (c Calendar) calcDay(twd time.Weekday) time.Time {
	cwd := c.present.Weekday()
	switch {
	case twd == cwd:
		return floorDay(c.present.Add(24 * 7 * time.Hour))
	case twd < cwd:
		return floorDay(c.present.Add(time.Duration((7-twd)*24) * time.Hour))
	default:
		return floorDay(c.present.Add(time.Duration((twd-cwd)*24) * time.Hour))
	}
}

/*
func calcMonth(s string, c Calendar) time.Time {
	twm := getMonthOfYear(s)
	cwm := c.present.Month()
	switch {
	case twm == cwm:
		return floorDay(c.present.AddDate(0, 12, 0))
	case twm < cwm:
		return floorDay(c.present.AddDate(0, int((12-cwm)+twm), 0))
	default:
		return floorDay(c.present.AddDate(0, int(twm-cwm), 0))
	}
}
*/

func (c Calendar) calcMonth(twm time.Month) time.Time {
	cwm := c.present.Month()
	switch {
	case twm == cwm:
		return floorMonth(floorDay(c.present.AddDate(0, 12, 0)))
	case twm < cwm:
		return floorMonth(floorDay(c.present.AddDate(0, int((12-cwm)+twm), 0)))
	default:
		return floorMonth(floorDay(c.present.AddDate(0, int(twm-cwm), 0)))
	}
}

func ceilDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 999999999, t.Location())
}

func floorDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func floorMonth(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

// NewCalendar returns a new calendar object
func NewCalendar(options ...func(*Calendar)) *Calendar {
	c := &Calendar{
		present: time.Now(),
	}
	for _, opt := range options {
		opt(c)
	}

	return c
}
