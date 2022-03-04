package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGatherSBOMRoots(t *testing.T) {

	predicate := func(filepath string) bool {
		return filepath == "Packages" || filepath == "Packages.lock"
	}

	t.Run("existing repository", func(t *testing.T) {
		got, err := GatherSBOMRoots("test-repository", predicate)
		require.Nil(t, err)
		require.ElementsMatch(t, []string{"test-repository", "test-repository/inner-dir/deepest-dir"}, got)
	})
	t.Run("non-existing repository", func(t *testing.T) {
		_, err := GatherSBOMRoots("/non-existing", predicate)
		require.NotNil(t, err)
	})
}
