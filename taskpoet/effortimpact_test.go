package taskpoet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriorityStrings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id          int
		description string
	}{
		{id: 0, description: "Unset"},
		{id: 4, description: "High Effort, Low Impact"},
	}

	for _, test := range tests {
		gotDescription := EffortImpactText(test.id)
		if gotDescription != test.description {
			t.Errorf("Expected %v but got %v", test.description, gotDescription)
		}
	}
}

func TestEIEmoji(t *testing.T) {
	require.Equal(t, "💀", EffortImpactAvoid.Emoji())
}
