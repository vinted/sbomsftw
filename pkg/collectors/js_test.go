package collectors

import (
	"context"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
)

func TestJSCollector(t *testing.T) {
	t.Run("bootstrap language files BOM roots correctly", func(t *testing.T) {
		executor := new(mockShellExecutor)
		executor.On("shellOut",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
			"pnpm install || npm install || yarn install").Return(nil)

		bomRoots := []string{
			"/tmp/some-random-dir/pnpm-lock.yaml",
			"/tmp/some-random-dir/package-lock.json",
			"/tmp/some-random-dir/inner-dir/yarn.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/package.json",
		}

		got := JS{executor: executor}.BootstrapLanguageFiles(context.Background(), bomRoots)
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
		executor.On("bomFromCdxgen", "/tmp/some-random-dir", "javascript", false).Return(new(cdx.BOM), nil)
		_, _ = JS{executor: executor}.GenerateBOM(context.Background(), bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		jsCollector := JS{}
		for _, f := range []string{"/opt/yarn.lock", "bower.json", "package.json", "pnpm-lock.yaml", "package-lock.json"} {
			assert.True(t, jsCollector.MatchLanguageFiles(false, f))
		}
		assert.False(t, jsCollector.MatchLanguageFiles(false, "/etc/passwd"))
		assert.False(t, jsCollector.MatchLanguageFiles(false, "/tmp/repo/node_modules/yarn.lock"))

		// Special case
		assert.True(t, jsCollector.MatchLanguageFiles(true, "/tmp/repo/node_modules"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "javascript collector", JS{}.String())
	})
}
