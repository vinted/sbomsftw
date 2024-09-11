package collectors

import (
	"context"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJVMCollector(t *testing.T) {
	t.Run("bootstrap language files correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/build.gradle",
			"/tmp/some-random-dir/build.gradle.kts",
			"/tmp/some-random-dir/inner-dir/pom.xml",
			"/tmp/some-random-dir/inner-dir/deepest-dir/build.sbt",
		}
		got := JVM{}.BootstrapLanguageFiles(context.Background(), bomRoots)
		assert.ElementsMatch(t, []string{
			"/tmp/some-random-dir",
			"/tmp/some-random-dir/inner-dir",
			"/tmp/some-random-dir/inner-dir/deepest-dir",
		}, got)
	})

	t.Run("regenerate BOM in multi module mode if single mode generation fails", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockShellExecutor)

		// Return an empty BOM on first run
		executor.On("bomFromCdxgen", bomRoot, "jvm", false).Return(new(cdx.BOM), nil)

		// Regenerate BOM a second time
		executor.On("bomFromCdxgen", bomRoot, "jvm", true).Return(new(cdx.BOM), nil)
		_, _ = JVM{executor: executor}.GenerateBOM(context.Background(), bomRoot)
		executor.AssertExpectations(t)
	})

	t.Run("use only single mode BOM if multi mode BOM generation fails", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockShellExecutor)

		components := []cdx.Component{{Name: "Dummy Component"}}
		bom := new(cdx.BOM)
		bom.Components = &components

		// Return a non-empty BOM the first time
		executor.On("bomFromCdxgen", bomRoot, "jvm", false).Return(bom, nil)
		// But return an empty BOM for the second time
		executor.On("bomFromCdxgen", bomRoot, "jvm", true).Return(new(cdx.BOM), nil)
		got, _ := JVM{executor: executor}.GenerateBOM(context.Background(), bomRoot)
		executor.AssertExpectations(t)
		assert.Equal(t, got, bom)
	})

	t.Run("collect BOMs correctly when multiple gradle build modes succeed", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockShellExecutor)

		firstBOMComponents := []cdx.Component{{PackageURL: "pkg:gem/addressable@2.4.0", Name: "addressable"}}
		firstBOM := new(cdx.BOM)
		firstBOM.Components = &firstBOMComponents

		secondBOMComponents := []cdx.Component{{PackageURL: "pkg:gem/addressable@2.4.1", Name: "addressable"}}
		secondBOM := new(cdx.BOM)
		secondBOM.Components = &secondBOMComponents

		// Return a non-empty BOM the first time
		executor.On("bomFromCdxgen", bomRoot, "jvm", false).Return(firstBOM, nil)
		// Return a non-empty BOM the second time as well
		executor.On("bomFromCdxgen", bomRoot, "jvm", true).Return(secondBOM, nil)

		got, _ := JVM{executor: executor}.GenerateBOM(context.Background(), bomRoot)
		executor.AssertExpectations(t)

		require.NotNil(t, got)
		require.NotNil(t, got.Components)

		var componentPURLs []string
		for _, c := range *got.Components {
			componentPURLs = append(componentPURLs, c.PackageURL)
		}

		assert.ElementsMatch(t, []string{
			"pkg:gem/addressable@2.4.0",
			"pkg:gem/addressable@2.4.1",
		}, componentPURLs)
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
