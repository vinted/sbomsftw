package collectors

import (
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
)

func TestRustCollector(t *testing.T) {
	t.Run("bootstrap language files correctly", func(t *testing.T) {
		languageFiles := []string{
			"/tmp/some-random-dir/Cargo.lock",
			"/tmp/some-random-dir/Cargo.toml",
			"/tmp/some-random-dir/inner-dir/Cargo.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/Cargo.toml",
		}

		got := Rust{}.BootstrapLanguageFiles(languageFiles)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockShellExecutor)
		executor.On("bomFromCdxgen", bomRoot, "rust", false).Return(new(cdx.BOM), nil)
		_, _ = Rust{executor: executor}.GenerateBOM(bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		rustCollector := Rust{}
		assert.True(t, rustCollector.MatchLanguageFiles(false, "Cargo.toml"))
		assert.True(t, rustCollector.MatchLanguageFiles(false, "/opt/Cargo.lock"))

		assert.False(t, rustCollector.MatchLanguageFiles(true, "Cargo.toml"))
		assert.False(t, rustCollector.MatchLanguageFiles(true, "Cargo.lock"))

		assert.False(t, rustCollector.MatchLanguageFiles(false, "/etc/passwd"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "rust collector", Rust{}.String())
	})
}
