package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJVMCollector(t *testing.T) {
	t.Run("bootstrap BOM roots correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/build.gradle",
			"/tmp/some-random-dir/build.gradle.kts",
			"/tmp/some-random-dir/inner-dir/pom.xml",
			"/tmp/some-random-dir/inner-dir/deepest-dir/build.sbt",
		}
		got := JVM{}.bootstrap(bomRoots)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockBOMBridge)
		executor.On("bomFromCdxgen", bomRoot, jvm).Return(new(cdx.BOM), nil)
		_, _ = JVM{executor: executor}.generateBOM(bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		jvmCollector := JVM{}
		for _, f := range []string{"/opt/pom.xml", "/opt/build.gradle", "build.gradle.kts", "sbt", "build.sbt"} {
			assert.True(t, jvmCollector.matchPredicate(false, f))
		}
		assert.False(t, jvmCollector.matchPredicate(false, "p0m.xml"))
		assert.False(t, jvmCollector.matchPredicate(true, "pom.xml"))
		assert.False(t, jvmCollector.matchPredicate(true, "build.gradle"))
	})

	t.Run("implement Stringer correctly", func(t *testing.T) {
		assert.Equal(t, "JVM - (Java/Kotlin/Scala/Groovy)", JVM{}.String())
	})
}
