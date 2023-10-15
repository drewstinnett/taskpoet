package taskpoet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAliases(t *testing.T) {
	require.Equal(t, []string{"jan"}, January.Aliases())
	require.Equal(t, []string{}, Synonym("Some-Never-Existent-Thing").Aliases())
}

func TestDescriptions(t *testing.T) {
	require.Equal(t, "Date for the specified month, starting at the beginning of the 1st day", January.Description())
	require.Equal(t, "", Synonym("Some-Never-Existent-Thing").Description())
}

func TestNth(t *testing.T) {
	c := NewCalendar(WithPresent(datePTR(time.Date(2023, 10, 5, 0, 0, 0, 0, time.Local))))
	tests := map[string]time.Time{
		"2days": time.Date(2023, 10, 7, 0, 0, 0, 0, time.Local),
		"jan":   time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
		"feb":   time.Date(2024, 2, 1, 0, 0, 0, 0, time.Local),
		"mar":   time.Date(2024, 3, 1, 0, 0, 0, 0, time.Local),
		"apr":   time.Date(2024, 4, 1, 0, 0, 0, 0, time.Local),
		"may":   time.Date(2024, 5, 1, 0, 0, 0, 0, time.Local),
		"jun":   time.Date(2024, 6, 1, 0, 0, 0, 0, time.Local),
		"jul":   time.Date(2024, 7, 1, 0, 0, 0, 0, time.Local),
		"aug":   time.Date(2024, 8, 1, 0, 0, 0, 0, time.Local),
		"sep":   time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local),
		"oct":   time.Date(2024, 10, 1, 0, 0, 0, 0, time.Local),
		"nov":   time.Date(2023, 11, 1, 0, 0, 0, 0, time.Local),
		"dec":   time.Date(2023, 12, 1, 0, 0, 0, 0, time.Local),
		"1st":   time.Date(2023, 11, 1, 0, 0, 0, 0, time.Local),
		"2nd":   time.Date(2023, 11, 2, 0, 0, 0, 0, time.Local),
		"3rd":   time.Date(2023, 11, 3, 0, 0, 0, 0, time.Local),
		"4th":   time.Date(2023, 11, 4, 0, 0, 0, 0, time.Local),
		"5th":   time.Date(2023, 11, 5, 0, 0, 0, 0, time.Local),
		"6th":   time.Date(2023, 10, 6, 0, 0, 0, 0, time.Local),
		"7th":   time.Date(2023, 10, 7, 0, 0, 0, 0, time.Local),
		"8th":   time.Date(2023, 10, 8, 0, 0, 0, 0, time.Local),
		"9th":   time.Date(2023, 10, 9, 0, 0, 0, 0, time.Local),
		"10th":  time.Date(2023, 10, 10, 0, 0, 0, 0, time.Local),
		"11th":  time.Date(2023, 10, 11, 0, 0, 0, 0, time.Local),
		"12th":  time.Date(2023, 10, 12, 0, 0, 0, 0, time.Local),
		"13th":  time.Date(2023, 10, 13, 0, 0, 0, 0, time.Local),
		"14th":  time.Date(2023, 10, 14, 0, 0, 0, 0, time.Local),
		"15th":  time.Date(2023, 10, 15, 0, 0, 0, 0, time.Local),
		"16th":  time.Date(2023, 10, 16, 0, 0, 0, 0, time.Local),
		"17th":  time.Date(2023, 10, 17, 0, 0, 0, 0, time.Local),
		"18th":  time.Date(2023, 10, 18, 0, 0, 0, 0, time.Local),
		"19th":  time.Date(2023, 10, 19, 0, 0, 0, 0, time.Local),
		"20th":  time.Date(2023, 10, 20, 0, 0, 0, 0, time.Local),
		"21st":  time.Date(2023, 10, 21, 0, 0, 0, 0, time.Local),
		"22nd":  time.Date(2023, 10, 22, 0, 0, 0, 0, time.Local),
		"23rd":  time.Date(2023, 10, 23, 0, 0, 0, 0, time.Local),
		"24th":  time.Date(2023, 10, 24, 0, 0, 0, 0, time.Local),
		"25th":  time.Date(2023, 10, 25, 0, 0, 0, 0, time.Local),
		"26th":  time.Date(2023, 10, 26, 0, 0, 0, 0, time.Local),
		"27th":  time.Date(2023, 10, 27, 0, 0, 0, 0, time.Local),
		"28th":  time.Date(2023, 10, 28, 0, 0, 0, 0, time.Local),
		"29th":  time.Date(2023, 10, 29, 0, 0, 0, 0, time.Local),
		"30th":  time.Date(2023, 10, 30, 0, 0, 0, 0, time.Local),
		"31st":  time.Date(2023, 10, 31, 0, 0, 0, 0, time.Local),
		"2ns":   time.Date(2023, 10, 5, 0, 0, 0, 2, time.Local), // Falling back to built in duration here
	}
	for given, expect := range tests {
		got, err := c.Date(given)
		require.NoError(t, err, given)
		require.Equal(t, expect, *got, given)
	}
}
