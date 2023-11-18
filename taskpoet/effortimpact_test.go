package taskpoet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriorityStrings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		given  EffortImpact
		expect string
	}{
		{given: EffortImpact(0), expect: "Unset"},
		{given: EffortImpact(4), expect: "High Effort, Low Impact"},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expect, tt.given.String())
	}
}

func TestEIEmoji(t *testing.T) {
	require.Equal(t, "💀", EffortImpactAvoid.Emoji())
}
