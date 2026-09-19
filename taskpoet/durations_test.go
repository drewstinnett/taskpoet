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
	const day = 24 * time.Hour
	tests := map[string]struct {
		given  time.Duration
		expect string
	}{
		"nothing":               {0, "0h"},
		"a couple hours":        {2 * time.Hour, "2h"},
		"almost a day":          {23*time.Hour + 59*time.Minute, "23h"},
		"a day":                 {day, "1d"},
		"a couple days":         {49 * time.Hour, "2d"},
		"almost a week":         {7*day - time.Minute, "6d"},
		"a week":                {7 * day, "1w"},
		"a couple weeks":        {15 * day, "2w"},
		"almost a month":        {30*day - time.Minute, "4w"},
		"a month":               {30 * day, "1M"},
		"a couple months":       {70 * day, "2M"},
		"almost a year":         {365*day - time.Minute, "12M"},
		"a year":                {365 * day, "1y"},
		"a year and change":     {400 * day, "1y"},
		"an imported old task":  {991 * day, "2y"},
		"a very old task":       {20 * 365 * day, "20y"},
		"negative hours":        {-2 * time.Hour, "-2h"},
		"negative days":         {-49 * time.Hour, "-2d"},
		"negative weeks":        {-15 * day, "-2w"},
		"negative months":       {-70 * day, "-2M"},
		"negative years":        {-400 * day, "-1y"},
		"negative just a month": {-30 * day, "-1M"},
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
	cal := NewCalendar(WithPresent(
		ptr(time.Date(2023, 10, 10, 8, 0, 0, 42, time.Local)), // This is a Tuesday

	))

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
	require.ErrorContains(t, err, "cannot make a date out of \"never-works\"")
	require.Nil(t, got)
}

func TestCalendarDateFormats(t *testing.T) {
	loc := time.FixedZone("test", -5*60*60)
	present := time.Date(2023, 10, 10, 8, 0, 0, 0, loc) // This is a Tuesday
	cal := NewCalendar(WithPresent(&present))

	tests := map[string]struct {
		in     string
		expect time.Time
	}{
		"synonym":              {"tomorrow", time.Date(2023, 10, 11, 0, 0, 0, 0, loc)},
		"taskwarrior days":     {"2d", present.Add(48 * time.Hour)},
		"taskwarrior weeks":    {"1w", present.Add(7 * 24 * time.Hour)},
		"go duration":          {"1.5h", present.Add(90 * time.Minute)},
		"go compound duration": {"1h30m", present.Add(90 * time.Minute)},
		"surrounding space":    {"  2d ", present.Add(48 * time.Hour)},
		"date":                 {"2024-05-01", time.Date(2024, 5, 1, 0, 0, 0, 0, loc)},
		"date and minutes":     {"2024-05-01 17:30", time.Date(2024, 5, 1, 17, 30, 0, 0, loc)},
		"date T minutes":       {"2024-05-01T17:30", time.Date(2024, 5, 1, 17, 30, 0, 0, loc)},
		"date and seconds":     {"2024-05-01 17:30:15", time.Date(2024, 5, 1, 17, 30, 15, 0, loc)},
		"date T seconds":       {"2024-05-01T17:30:15", time.Date(2024, 5, 1, 17, 30, 15, 0, loc)},
		"rfc3339 keeps zone":   {"2024-05-01T17:30:00+02:00", time.Date(2024, 5, 1, 15, 30, 0, 0, time.UTC)},
		"compact date":         {"20240501", time.Date(2024, 5, 1, 0, 0, 0, 0, loc)},
		"taskwarrior is utc":   {"20240501T173000Z", time.Date(2024, 5, 1, 17, 30, 0, 0, time.UTC)},
		"leap day":             {"2024-02-29", time.Date(2024, 2, 29, 0, 0, 0, 0, loc)},
		"end of year":          {"2023-12-31 23:59", time.Date(2023, 12, 31, 23, 59, 0, 0, loc)},
		"a date in the past":   {"2001-09-09", time.Date(2001, 9, 9, 0, 0, 0, 0, loc)},
		"no leading zeros":     {"2024-1-5", time.Date(2024, 1, 5, 0, 0, 0, 0, loc)},
	}
	for desc, tt := range tests {
		t.Run(desc, func(t *testing.T) {
			got, err := cal.Date(tt.in)
			require.NoError(t, err)
			require.True(t, tt.expect.Equal(*got), "want %v, got %v", tt.expect, *got)
		})
	}

	// These used to be read as their first word, e.g. "3 days ago" as 3 days ahead
	for _, bad := range []string{"", "nonsense", "3 days ago", "2d ago", "2d 4h", "2024-13-01", "2023-02-29", "2024-05-01 25:00", "05/01/2024"} {
		t.Run("invalid "+bad, func(t *testing.T) {
			_, err := cal.Date(bad)
			require.Error(t, err)
			require.Contains(t, err.Error(), "cannot make a date out of")
		})
	}
}
