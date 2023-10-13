package taskpoet

import (
	"testing"

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
