package taskpoet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultNewCurator(t *testing.T) {
	c := NewCurator()
	require.NotNil(t, c)
	require.NotPanics(t, func() {
		c.Weigh(*MustNewTask("some task"))
	})
}

func TestNewCuratorWithWeights(t *testing.T) {
	c := NewCurator(WithWeights(
		weightMap{
			"foo": func(t Task) (float64, int, string) { return float64(1), 1, "" },
			"bar": func(t Task) (float64, int, string) { return float64(1), 1, "" },
		},
	))
	require.NotNil(t, c)
	require.Equal(t, float64(2), c.Weigh(*MustNewTask("some thing")))
}
