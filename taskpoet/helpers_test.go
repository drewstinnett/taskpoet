package taskpoet_test

import (
	"testing"

	"github.com/drewstinnett/taskpoet/taskpoet"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		duration string
		seconds  float64
	}{
		{"0", 0},
		{"2h", 7200},
		{"-2h", -7200},
		{".5h", 1800},
		{"02h", 7200},
		{"24h", 86400},
		{"1d", 86400},
		{"2h30m", 9000},
	}

	for _, test := range tests {
		dur, _ := taskpoet.ParseDuration(test.duration)
		if dur.Seconds() != test.seconds {
			t.Errorf("Expected %v seconds from duration %v...but got %v", test.seconds, test.duration, dur.Seconds())
		}

	}
}

func TestParseInvalidDuration(t *testing.T) {
	tests := []struct {
		duration string
	}{
		{""},
		{"5"},
		{".ah"},
		{".s"},
		{"-.s"},
	}

	for _, test := range tests {
		_, err := taskpoet.ParseDuration(test.duration)
		if err == nil {
			t.Errorf("Did not return an error when parsing the invalid duration: '%v'", test.duration)
		}

	}
}

func TestUniqueSlices(t *testing.T) {
	nodups := []string{"a", "b", "c"}
	dups := []string{"c", "b", "c"}

	if !taskpoet.CheckUniqueStringSlice(nodups) {
		t.Error("Checking slice with no duplicates detected duplicates")
	}

	if taskpoet.CheckUniqueStringSlice(dups) {
		t.Error("Checking a slice with dups did not detect dups")
	}
}
