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

func TestCalendarSynonyms(t *testing.T) {
	present := time.Date(1978, 7, 16, 8, 0, 0, 42, time.Local)
	cal := NewCalendar(WithPresent(&present))
	tests := map[string]time.Time{
		"now":       time.Date(1978, 7, 16, 8, 0, 0, 42, time.Local),
		"today":     time.Date(1978, 7, 16, 0, 0, 0, 0, time.Local),
		"tomorrow":  time.Date(1978, 7, 17, 0, 0, 0, 0, time.Local),
		"yesterday": time.Date(1978, 7, 15, 0, 0, 0, 0, time.Local),
		"eod":       time.Date(1978, 7, 16, 23, 59, 59, 999, time.Local),
	}
	for given, expect := range tests {
		got, err := cal.Synonym(given)
		require.NoError(t, err)
		require.Equal(t, expect, *got)
	}

	got, err := cal.Synonym("never-exists")
	require.Nil(t, got)
	require.Error(t, err)
	require.EqualError(t, err, "unknown synonym: never-exists")
}
