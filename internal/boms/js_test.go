package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJSCollector(t *testing.T) {
	t.Run("bootstrap BOM roots correctly", func(t *testing.T) {
		executor := new(mockCLIExecutor)
		executor.On("shellOut",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
			"pnpm install || npm install || yarn install").Return("👌", nil)

		bomRoots := []string{
			"/tmp/some-random-dir/pnpm-lock.yaml",
			"/tmp/some-random-dir/package-lock.json",
			"/tmp/some-random-dir/inner-dir/yarn.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/package.json",
		}

		got := JS{executor: executor}.bootstrap(bomRoots)
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
		executor.On("bomFromCdxgen", bomRoot, javascript).Return(new(cdx.BOM), nil)
		_, _ = JS{executor: executor}.generateBOM(bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		jsCollector := JS{}
		for _, f := range []string{"yarn.lock", "bower.json", "package.json", "pnpm-lock.yaml", "package-lock.json"} {
			assert.True(t, jsCollector.matchPredicate(false, f))
		}
		assert.False(t, jsCollector.matchPredicate(false, "/etc/passwd"))
		assert.False(t, jsCollector.matchPredicate(false, "/tmp/repo/node_modules/yarn.lock"))

		//Special case
		assert.True(t, jsCollector.matchPredicate(true, "/tmp/repo/node_modules"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "JS/TS-JS", JS{}.String())
	})
}
