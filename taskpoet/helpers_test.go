package taskpoet_test

import (
	"testing"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		duration string
		seconds  float64
	}{
		{duration: "0", seconds: 0},
		{duration: "2h", seconds: 7200},
		{duration: "-2h", seconds: -7200},
		{duration: ".5h", seconds: 1800},
		{duration: "02h", seconds: 7200},
		{duration: "24h", seconds: 86400},
		{duration: "1d", seconds: 86400},
		{duration: "2h30m", seconds: 9000},
	}

	for _, test := range tests {
		dur, _ := taskpoet.ParseDuration(test.duration)
		require.Equal(t, test.seconds, dur.Seconds())
	}
}

func TestParseInvalidDuration(t *testing.T) {
	for _, tt := range []string{"", "5", ".ah", ".s", "-.s"} {
		_, err := taskpoet.ParseDuration(tt)
		require.Error(t, err)
	}
}

func TestUniqueSlices(t *testing.T) {
	require.True(t, taskpoet.CheckUniqueStringSlice([]string{"a", "b", "c"}))
	require.False(t, taskpoet.CheckUniqueStringSlice([]string{"c", "b", "c"}))
}
