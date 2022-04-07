package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	fp "path/filepath"
)

type JVM struct {
	executor CLIExecutor
}

func NewJVMCollector() JVM {
	return JVM{executor: defaultCLIExecutor{}}
}

//matchPredicate Implements BOMCollector interface
func (j JVM) matchPredicate(isDir bool, filepath string) bool {
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
	return "JVM - (Java/Kotlin/Scala/Groovy)"
}

func (j JVM) generateBOM(bomRoot string) (*cdx.BOM, error) {
	return j.executor.bomFromCdxgen(bomRoot, jvm)
}

func (j JVM) bootstrap(bomRoots []string) []string {
	return squashRoots(bomRoots)
}
