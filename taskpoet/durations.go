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
type Synonym func() *time.Time

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
/*
   monday, tuesday, … - Local date for the specified day, after today, with time 00:00:00. Can be shortened, e.g. mon, tue 2.6.0 Can be capitalized, e.g. Monday, Tue
   january, february, … - Local date for the specified month, 1st day, with time 00:00:00. Can be shortened, e.g. jan, feb. 2.6.0 Can be capitalized, e.g. January, Feb.
   later, someday - Local 2038-01-18, with time 00:00:00. A date far away, with semantically meaningful to GTD users.
   soy - Local date for the next year, January 1st, with time 00:00:00.
   eoy - Local date for this year, December 31st, with time 00:00:00.
   soq - Local date for the start of the next quarter (January, April, July, October), 1st, with time 00:00:00.
   eoq - Local date for the end of the current quarter (March, June, September, December), last day of the month, with time 23:59:59.
   som - Local date for the 1st day of the next month, with time 00:00:00.
   socm - Local date for the 1st day of the current month, with time 00:00:00.
   eom, eocm - Local date for the last day of the current month, with time 23:59:59.
   sow - Local date for the next Sunday, with time 00:00:00.
   socw - Local date for the last Sunday, with time 00:00:00.
   eow, eocw - Local date for the end of the week, Saturday night, with time 00:00:00.
   soww - Local date for the start of the work week, next Monday, with time 00:00:00.
   eoww - Local date for the end of the work week, Friday night, with time 23:59:59.
   1st, 2nd, … - Local date for the next Nth day, with time 00:00:00.
   goodfriday - Local date for the next Good Friday, with time 00:00:00.
   easter - Local date for the next Easter Sunday, with time 00:00:00.
   eastermonday - Local date for the next Easter Monday, with time 00:00:00.
   ascension - Local date for the next Ascension (39 days after Easter Sunday), with time 00:00:00.
   pentecost - Local date for the next Pentecost (40 days after Easter Sunday), with time 00:00:00.
   midsommar - Local date for the Saturday after June 20th, with time 00:00:00. Swedish.
   midsommarafton - Local date for the Friday after June 19th, with time 00:00:00. Swedish.
*/
func (c Calendar) Synonym(s string) (*time.Time, error) {
	switch s {
	case "now":
		return &c.present, nil
	case "today":
		year, month, day := c.present.Date()
		d := time.Date(year, month, day, 0, 0, 0, 0, c.present.Location())
		return &d, nil
	case "tomorrow", "sod":
		year, month, day := c.present.Add(24 * time.Hour).Date()
		d := time.Date(year, month, day, 0, 0, 0, 0, c.present.Location())
		return &d, nil
	case "yesterday":
		year, month, day := c.present.Add(-24 * time.Hour).Date()
		d := time.Date(year, month, day, 0, 0, 0, 0, c.present.Location())
		return &d, nil
	case "eod":
		year, month, day := c.present.Date()
		d := time.Date(year, month, day, 23, 59, 59, 999, c.present.Location())
		return &d, nil
	case "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday":
		twd := daysOfWeek[s]
		cwd := c.present.Weekday()
		if twd == cwd {
			// year month, day := c.present.Add(24 * 7 * time.Hour).Date()
			next := c.present.Add(24 * 7 * time.Hour)
			year, month, day := next.Date()
			d := time.Date(year, month, day, 0, 0, 0, 0, c.present.Location())
			return &d, nil
		}
		_ = c.present.Weekday()

	default:
		return nil, fmt.Errorf("unknown synonym: %v", s)
	}
	return nil, nil
}

var daysOfWeek = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
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
