package collectors

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	fp "path/filepath"
)

type JVM struct {
	executor ShellExecutor
}

func NewJVMCollector() JVM {
	return JVM{executor: DefaultShellExecutor{}}
}

//MatchLanguageFiles Implements LanguageCollector interface
func (j JVM) MatchLanguageFiles(isDir bool, filepath string) bool {
	if isDir {
		return false
	}
	for _, f := range []string{"pom.xml", "build.gradle", "build.gradle.kts", "sbt", "build.sbt"} {
		if fp.Base(filepath) == f {
			return true
		}
	}
	return false
}

func (j JVM) String() string {
	return "jvm collector"
}

func (j JVM) GenerateBOM(bomRoot string) (*cdx.BOM, error) {
	const language = "jvm"
	return j.executor.bomFromCdxgen(bomRoot, language)
}

func (j JVM) BootstrapLanguageFiles(bomRoots []string) []string {
	return bomRoots
}
