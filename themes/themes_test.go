package themes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotNil(t *testing.T) {
	require.NotNil(t, New())
}
