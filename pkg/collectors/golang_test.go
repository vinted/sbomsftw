package collectors

import (
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
)

func TestGolangCollector(t *testing.T) {
	t.Run("Bootstrap language files correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/go.mod",
			"/tmp/some-random-dir/go.sum",
			"/tmp/some-random-dir/inner-dir/go.mod",
			"/tmp/some-random-dir/inner-dir/deepest-dir/Gopkg.lock",
		}
		got := Golang{}.BootstrapLanguageFiles(bomRoots)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("match correct package files", func(t *testing.T) {
		golangCollector := Golang{}
		for _, f := range []string{"/opt/go.mod", "go.sum", "Gopkg.lock"} {
			assert.True(t, golangCollector.MatchLanguageFiles(false, f))
		}
		assert.False(t, golangCollector.MatchLanguageFiles(false, "/etc/passwd"))
		assert.False(t, golangCollector.MatchLanguageFiles(false, "/tmp/test-repo/vendor/go.sum"))
		assert.False(t, golangCollector.MatchLanguageFiles(false, "/tmp/test-repo/inner-dir/vendor/go.mod"))
		assert.False(t, golangCollector.MatchLanguageFiles(true, "/tmp/test-repo/vendor"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "golang collector", Golang{}.String())
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockShellExecutor)
		executor.On("bomFromCdxgen", "/tmp/some-random-dir", "golang", false).Return(new(cdx.BOM), nil)
		_, _ = Golang{executor: executor}.GenerateBOM(bomRoot)
		executor.AssertExpectations(t)
	})
}
