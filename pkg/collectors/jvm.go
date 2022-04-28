package collectors

import (
	fp "path/filepath"

	log "github.com/sirupsen/logrus"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
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
	for _, f := range []string{"pom.xml", "gradlew", "sbt", "build.sbt"} {
		if fp.Base(filepath) == f {
			return true
		}
	}
	return false
}

func (j JVM) String() string {
	return "jvm collector"
}

//GenerateBOM implements LanguageCollector interface
func (j JVM) GenerateBOM(bomRoot string) (*cdx.BOM, error) {

	isBOMEmpty := func(bom *cdx.BOM) bool {
		return bom == nil || bom.Components == nil || len(*bom.Components) == 0
	}

	const language = "jvm"
	singleModeBom, err := j.executor.bomFromCdxgen(bomRoot, language, false)
	if err != nil || isBOMEmpty(singleModeBom) {
		log.WithFields(log.Fields{
			"collector":       j,
			"collection path": bomRoot,
		}).Debugf("can't collect BOMs, retrying collection in multi module mode (gradle)")
		return j.executor.bomFromCdxgen(bomRoot, language, true)
	}

	multiModeBom, err := j.executor.bomFromCdxgen(bomRoot, language, true)
	if err != nil || isBOMEmpty(multiModeBom) {
		log.WithFields(log.Fields{
			"collector":       j,
			"collection path": bomRoot,
		}).Debugf("can't collect BOMs in multi module mode (gradle), using single module BOM")
		return singleModeBom, nil
	}
	return bomtools.MergeBoms(singleModeBom, multiModeBom)
}

//BootstrapLanguageFiles implements LanguageCollector interface
func (j JVM) BootstrapLanguageFiles(bomRoots []string) []string {
	const bootstrapCmd = "./gradlew"
	for dir, files := range SplitPaths(bomRoots) {
		for _, f := range files {
			if f == "gradlew" {
				log.WithFields(log.Fields{
					"collector":       j,
					"collection path": dir,
				}).Debug("priming gradle cache")

				if err := j.executor.shellOut(dir, bootstrapCmd); err != nil {
					log.WithFields(log.Fields{
						"error":           err,
						"collector":       j,
						"collection path": dir,
					}).Debug("can't prime gradle cache")
					continue
				}
				log.WithFields(log.Fields{
					"collector":       j,
					"collection path": dir,
				}).Debug("gradle cache primed successfuly")

			}
		}
	}
	return SquashToDirs(bomRoots)
}
