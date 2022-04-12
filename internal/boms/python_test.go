package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPythonCollector(t *testing.T) {
	t.Run("bootstrap BOM roots correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/requirements.txt",
			"/tmp/some-random-dir/setup.py",
			"/tmp/some-random-dir/inner-dir/Pipfile.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/poetry.lock",
		}
		got := Python{}.bootstrap(bomRoots)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockBOMBridge)
		executor.On("bomFromCdxgen", bomRoot, python).Return(new(cdx.BOM), nil)
		_, _ = Python{executor: executor}.generateBOM(bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		golangCollector := Python{}
		for _, f := range []string{"setup.py", "requirements.txt", "Pipfile.lock", "poetry.lock"} {
			assert.True(t, golangCollector.matchPredicate(false, f))
		}
		for _, f := range []string{"setup.py", "requirements.txt", "Pipfile.lock", "poetry.lock"} {
			assert.False(t, golangCollector.matchPredicate(true, f))
		}
		assert.False(t, golangCollector.matchPredicate(false, "/etc/passwd"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "Python collector", Python{}.String())
	})
}
