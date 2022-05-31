package collectors

import (
	"context"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
)

func TestRubyCollector(t *testing.T) {
	t.Run("bootstrap language files correctly", func(t *testing.T) {
		executor := new(mockShellExecutor)
		executor.On("shellOut",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
			"bundler install ||  bundler _1.9_ install || bundler _1.17.3_ install").Return(nil)

		languageFiles := []string{
			"/tmp/some-random-dir/Gemfile",
			"/tmp/some-random-dir/Gemfile.lock",
			"/tmp/some-random-dir/inner-dir/Gemfile.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/Gemfile",
		}

		got := Ruby{executor: executor}.BootstrapLanguageFiles(context.Background(), languageFiles)
		executor.AssertExpectations(t)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockShellExecutor)
		executor.On("bomFromCdxgen", bomRoot, "ruby", false).Return(new(cdx.BOM), nil)
		_, _ = Ruby{executor: executor}.GenerateBOM(context.Background(), bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		rubyCollector := Ruby{}
		assert.True(t, rubyCollector.MatchLanguageFiles(false, "Gemfile"))
		assert.True(t, rubyCollector.MatchLanguageFiles(false, "/opt/Gemfile.lock"))
		assert.False(t, rubyCollector.MatchLanguageFiles(false, "/etc/passwd"))
		assert.False(t, rubyCollector.MatchLanguageFiles(true, "Gemfile"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "ruby collector", Ruby{}.String())
	})
}
