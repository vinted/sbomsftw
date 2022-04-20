package collectors

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJVMCollector(t *testing.T) {
	t.Run("BootstrapLanguageFiles BOM roots correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/build.gradle",
			"/tmp/some-random-dir/build.gradle.kts",
			"/tmp/some-random-dir/inner-dir/pom.xml",
			"/tmp/some-random-dir/inner-dir/deepest-dir/build.sbt",
		}
		got := JVM{}.BootstrapLanguageFiles(bomRoots)
		assert.Equal(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockBOMBridge)
		executor.On("bomFromCdxgen", bomRoot, "jvm", true).Return(new(cdx.BOM), nil)
		_, _ = JVM{executor: executor}.GenerateBOM(bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		jvmCollector := JVM{}
		for _, f := range []string{"/opt/pom.xml", "/opt/gradlew", "gradlew", "sbt", "build.sbt"} {
			assert.True(t, jvmCollector.MatchLanguageFiles(false, f))
		}
		assert.False(t, jvmCollector.MatchLanguageFiles(false, "p0m.xml"))
		assert.False(t, jvmCollector.MatchLanguageFiles(true, "pom.xml"))
		assert.False(t, jvmCollector.MatchLanguageFiles(false, "build.gradle"))
		assert.False(t, jvmCollector.MatchLanguageFiles(false, "build.gradle.kts"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "jvm collector", JVM{}.String())
	})
}
