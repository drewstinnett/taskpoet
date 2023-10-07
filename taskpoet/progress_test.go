package taskpoet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProgress(t *testing.T) {
	require.NotPanics(t, func() {
		NewProgressBar(WithStatusChannel(make(chan ProgressStatus)))
	})
}
