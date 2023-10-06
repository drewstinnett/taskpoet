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
