package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRubyCollector(t *testing.T) {
	t.Run("bootstrap BOM roots correctly", func(t *testing.T) {
		executor := new(mockCLIExecutor)
		executor.On("shellOut",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
			"bundler install ||  bundler _1.9_ install || bundler _1.17.3_ install").Return("👌", nil)

		bomRoots := []string{
			"/tmp/some-random-dir/Gemfile",
			"/tmp/some-random-dir/Gemfile.lock",
			"/tmp/some-random-dir/inner-dir/Gemfile.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/Gemfile",
		}

		got := Ruby{executor: executor}.bootstrap(bomRoots)
		executor.AssertExpectations(t)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockCLIExecutor)
		executor.On("bomFromCdxgen", bomRoot, ruby).Return(new(cdx.BOM), nil)
		_, _ = Ruby{executor: executor}.generateBOM(bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		rubyCollector := Ruby{}
		assert.True(t, rubyCollector.matchPredicate(false, "Gemfile"))
		assert.True(t, rubyCollector.matchPredicate(false, "Gemfile.lock"))
		assert.False(t, rubyCollector.matchPredicate(false, "/etc/passwd"))
		assert.False(t, rubyCollector.matchPredicate(true, "Gemfile"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "Ruby-Bundler", Ruby{}.String())
	})
}
