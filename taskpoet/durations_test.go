package taskpoet

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDurations(t *testing.T) {
	tests := map[string]time.Duration{
		"5 s":        time.Second * 5,
		"5s":         time.Second * 5,
		"second":     time.Second * 1,
		"10 minute":  time.Minute * 10,
		"minute":     time.Minute * 1,
		"2 hours":    time.Hour * 2,
		"daily":      time.Hour * 24,
		"2 days":     time.Hour * 48,
		"2 weeks":    time.Hour * 336,
		"monthly":    time.Hour * 720,
		"quarterly":  time.Hour * 2184,
		"2q":         time.Hour * 4368,
		"semiannual": time.Hour * 4320,
		"yearly":     time.Hour * 8760,
	}
	for given, expect := range tests {
		got, err := parseDuration(given)
		require.NoError(t, err)
		require.Equal(t, expect, *got)
	}
}

func TestDurtionErrors(t *testing.T) {
	tests := map[string]string{
		"5nothing": "invalid unit: nothing",
		"":         "duration must not be an empty string",
	}
	for given, expect := range tests {
		got, err := parseDuration(given)
		require.Error(t, err)
		require.Nil(t, got)
		require.EqualError(t, errors.New(expect), err.Error())
	}
}

func TestNewCalendar(t *testing.T) {
	now := time.Now()
	got := NewCalendar(WithPresent(&now))
	require.NotNil(t, got)
}

func TestShortDurations(t *testing.T) {
	require.Equal(t, "2h", shortDuration(2*time.Hour), "simple-positive")
	require.Equal(t, "-2h", shortDuration(-2*time.Hour), "simple-negative")
	tests := map[string]struct {
		given  time.Duration
		expect string
	}{
		"a couple hours": {
			given:  time.Hour * 2,
			expect: "2h",
		},
		"a couple hours ago": {
			given:  -time.Hour * 2,
			expect: "-2h",
		},
		"a couple days": {
			given:  time.Hour * 49,
			expect: "2d",
		},
		"a couple days ago": {
			given:  -time.Hour * 49,
			expect: "-2d",
		},
		"a couple weeks": {
			given:  time.Hour * 24 * 15,
			expect: "2w",
		},
		"a couple weeks ago": {
			given:  -time.Hour * 24 * 15,
			expect: "-2w",
		},
		"a couple months": {
			given:  time.Hour * 24 * 7 * 70,
			expect: "2M",
		},
		"a couple months ago": {
			given:  -time.Hour * 24 * 7 * 70,
			expect: "-2M",
		},
		"a year": {
			given:  time.Hour * 24 * 7 * 30 * 400,
			expect: "1y",
		},
		"a year ago": {
			given:  -time.Hour * 24 * 7 * 30 * 400,
			expect: "-1y",
		},
	}
	for desc, tt := range tests {
		require.Equal(t, tt.expect, shortDuration(tt.given), desc)
	}
}

func TestCalendarMonth(t *testing.T) {
	present := time.Date(2023, 10, 1, 0, 0, 0, 42, time.Local) // This is a October
	cal := NewCalendar(WithPresent(&present))

	tests := map[string]struct {
		given  string
		expect time.Time
	}{
		"same-month": {
			given:  "october",
			expect: time.Date(2024, 10, 1, 0, 0, 0, 0, time.Local),
		},
		"same-month-abbr": {
			given:  "oct",
			expect: time.Date(2024, 10, 1, 0, 0, 0, 0, time.Local),
		},
		"previous-month": {
			given:  "sep",
			expect: time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local),
		},
		"next-month": {
			given:  "dec",
			expect: time.Date(2023, 12, 1, 0, 0, 0, 0, time.Local),
		},
		"next-month-rollover": {
			given:  "jan",
			expect: time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
		},
	}
	for desc, tt := range tests {
		got, err := cal.Synonym(tt.given)
		require.NoError(t, err, desc)
		require.Equal(t, tt.expect, got, desc)
	}
}

func TestCalendarWeekday(t *testing.T) {
	present := time.Date(2023, 10, 10, 8, 0, 0, 42, time.Local) // This is a Tuesday
	cal := NewCalendar(WithPresent(&present))

	tests := map[string]struct {
		given    string
		expect   time.Time
		calendar *Calendar
	}{
		"same-day": {
			given:  "tuesday",
			expect: time.Date(2023, 10, 17, 0, 0, 0, 0, time.Local),
		},
		"same-abbr": {
			given:  "tue",
			expect: time.Date(2023, 10, 17, 0, 0, 0, 0, time.Local),
		},
		"previous-day": {
			given:  "monday",
			expect: time.Date(2023, 10, 16, 0, 0, 0, 0, time.Local),
		},
		"next-day": {
			given:  "wednesday",
			expect: time.Date(2023, 10, 11, 0, 0, 0, 0, time.Local),
		},
		"sow-under-week": {
			given:  "sow",
			expect: time.Date(2023, 10, 15, 0, 0, 0, 0, time.Local),
		},
		"socw": {
			given:  "socw",
			expect: time.Date(2023, 10, 8, 0, 0, 0, 0, time.Local),
		},
	}

	for desc, tt := range tests {
		got, err := cal.Synonym(tt.given)
		require.NoError(t, err, desc)
		require.Equal(t, tt.expect, got, desc)
	}
}

func TestGenericCalendarSynonyms(t *testing.T) {
	wednesday := time.Date(2023, 10, 11, 0, 0, 0, 0, time.Local)
	som := time.Date(2023, 10, 1, 0, 0, 0, 0, time.Local) // 1st of month
	tests := map[string]struct {
		given    string
		expect   time.Time
		calendar Calendar
	}{
		"start-of-work-week": {
			given:    "soww",
			expect:   time.Date(2023, 10, 16, 0, 0, 0, 0, time.Local),
			calendar: *NewCalendar(WithPresent(&wednesday)),
		},
		"end-of-work-week": {
			given:    "eoww",
			expect:   time.Date(2023, 10, 13, 23, 59, 59, 999999999, time.Local),
			calendar: *NewCalendar(WithPresent(&wednesday)),
		},
		"next-1st": {
			given:    "1st",
			expect:   time.Date(2023, 11, 1, 0, 0, 0, 0, time.Local),
			calendar: *NewCalendar(WithPresent(&som)),
		},
		"this-2nd": {
			given:    "2nd",
			expect:   time.Date(2023, 10, 2, 0, 0, 0, 0, time.Local),
			calendar: *NewCalendar(WithPresent(&som)),
		},
	}
	for desc, tt := range tests {
		got, err := tt.calendar.Synonym(tt.given)
		require.NoError(t, err, desc)
		require.Equal(t, tt.expect, got, desc)
	}
}

func TestCalendarSynonyms(t *testing.T) {
	present := time.Date(1978, 7, 16, 8, 0, 0, 42, time.Local)
	cal := NewCalendar(WithPresent(&present))
	tests := map[string]time.Time{
		"now":       time.Date(1978, 7, 16, 8, 0, 0, 42, time.Local),
		"today":     time.Date(1978, 7, 16, 0, 0, 0, 0, time.Local),
		"tomorrow":  time.Date(1978, 7, 17, 0, 0, 0, 0, time.Local),
		"sod":       time.Date(1978, 7, 17, 0, 0, 0, 0, time.Local),
		"yesterday": time.Date(1978, 7, 15, 0, 0, 0, 0, time.Local),
		"eod":       time.Date(1978, 7, 16, 23, 59, 59, 999999999, time.Local),
		"someday":   time.Date(292277026596, time.December, 4, 10, 30, 7, 0, time.Local),
		"later":     time.Date(292277026596, time.December, 4, 10, 30, 7, 0, time.Local),
		"soy":       time.Date(1979, 1, 1, 0, 0, 0, 0, time.Local),
		"eoy":       time.Date(1978, 12, 31, 0, 0, 0, 0, time.Local),
		"som":       time.Date(1978, 8, 1, 0, 0, 0, 0, time.Local),
		"socm":      time.Date(1978, 7, 1, 0, 0, 0, 0, time.Local),
		"eom":       time.Date(1978, 7, 31, 23, 59, 59, 999999999, time.Local),
		"eocm":      time.Date(1978, 7, 31, 23, 59, 59, 999999999, time.Local),
		"eow":       time.Date(1978, 7, 22, 23, 59, 59, 999999999, time.Local),
		"eocw":      time.Date(1978, 7, 22, 23, 59, 59, 999999999, time.Local),
	}
	for given, expect := range tests {
		got, err := cal.Synonym(given)
		require.NoError(t, err, given)
		require.Equal(t, expect, got, given)
	}

	got, err := cal.Synonym("never-exists")
	require.Equal(t, got, time.Time{})
	require.Error(t, err)
	require.EqualError(t, err, "unknown synonym: never-exists")
}

func TestSynonymWithAlias(t *testing.T) {
	s, err := synonymWithAlias("january")
	require.NoError(t, err)
	require.Equal(t, &January, s)

	s, err = synonymWithAlias("jan")
	require.NoError(t, err)
	require.Equal(t, &January, s)
}

func TestCalendarDate(t *testing.T) {
	present := time.Date(2023, 10, 1, 0, 0, 0, 42, time.Local) // This is a October
	c := NewCalendar(WithPresent(&present))

	got, err := c.Date("now")
	require.NoError(t, err)
	require.Equal(t, present, *got)

	got, err = c.Date("5h")
	require.NoError(t, err)
	require.Equal(t, time.Date(2023, 10, 1, 5, 0, 0, 42, time.Local), *got)

	got, err = c.Date("never-works")
	require.Error(t, err)
	require.EqualError(t, err, "time: invalid duration \"never-works\"")
	require.Nil(t, got)
}
