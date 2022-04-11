package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRustCollector(t *testing.T) {
	t.Run("bootstrap BOM roots correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/Cargo.lock",
			"/tmp/some-random-dir/Cargo.toml",
			"/tmp/some-random-dir/inner-dir/Cargo.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/Cargo.toml",
		}
		got := Rust{}.bootstrap(bomRoots)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockCLIExecutor)
		executor.On("bomFromCdxgen", bomRoot, rust).Return(new(cdx.BOM), nil)
		_, _ = Rust{executor: executor}.generateBOM(bomRoot)
		executor.AssertExpectations(t)
	})
	t.Run("match correct package files", func(t *testing.T) {
		rustCollector := Rust{}
		assert.True(t, rustCollector.matchPredicate(false, "Cargo.toml"))
		assert.True(t, rustCollector.matchPredicate(false, "Cargo.lock"))

		assert.False(t, rustCollector.matchPredicate(true, "Cargo.toml"))
		assert.False(t, rustCollector.matchPredicate(true, "Cargo.lock"))

		assert.False(t, rustCollector.matchPredicate(false, "/etc/passwd"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "Rust collector", Rust{}.String())
	})
}
