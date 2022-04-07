package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGolangCollector(t *testing.T) {
	t.Run("bootstrap BOM roots correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/go.mod",
			"/tmp/some-random-dir/go.sum",
			"/tmp/some-random-dir/inner-dir/go.mod",
			"/tmp/some-random-dir/inner-dir/deepest-dir/Gopkg.lock",
		}
		got := Golang{}.bootstrap(bomRoots)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockCLIExecutor)
		executor.On("bomFromCdxgen", bomRoot, golang).Return(new(cdx.BOM), nil)
		_, _ = Golang{executor: executor}.generateBOM(bomRoot)
		executor.AssertExpectations(t)
	})
	t.Run("match correct package files", func(t *testing.T) {
		golangCollector := Golang{}
		for _, f := range []string{"go.mod", "go.sum", "Gopkg.lock"} {
			assert.True(t, golangCollector.matchPredicate(false, f))
		}
		assert.False(t, golangCollector.matchPredicate(false, "/etc/passwd"))
		assert.False(t, golangCollector.matchPredicate(false, "/tmp/test-repo/vendor/go.sum"))
		assert.False(t, golangCollector.matchPredicate(false, "/tmp/test-repo/inner-dir/vendor/go.mod"))
		assert.False(t, golangCollector.matchPredicate(true, "/tmp/test-repo/vendor"))
	})
	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "Golang collector", Golang{}.String())
	})
}
