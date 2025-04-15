package collectors

import (
	"context"
	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
	"github.com/vinted/sbomsftw/pkg/bomtools"
	fp "path/filepath"
)

type JVM struct {
	executor shellExecutor
}

func NewJVMCollector() JVM {
	return JVM{
		executor: defaultShellExecutor{},
	}
}

// MatchLanguageFiles Implements LanguageCollector interface
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

// GenerateBOM implements LanguageCollector interface
func (j JVM) GenerateBOM(ctx context.Context, bomRoot string) (*cdx.BOM, error) {
	isBOMEmpty := func(bom *cdx.BOM) bool {
		return bom == nil || bom.Components == nil || len(*bom.Components) == 0
	}

	const language = "jvm"
	singleModeBom, err := j.executor.bomFromCdxgen(ctx, bomRoot, language, false)

	if err != nil || isBOMEmpty(singleModeBom) {
		log.WithFields(log.Fields{
			"collector":       j,
			"collection path": bomRoot,
		}).Debugf("can't collect BOMs, retrying collection in multi module mode (gradle)")

		return j.executor.bomFromCdxgen(ctx, bomRoot, language, true)
	}

	multiModeBom, err := j.executor.bomFromCdxgen(ctx, bomRoot, language, true)
	if err != nil || isBOMEmpty(multiModeBom) {
		log.WithFields(log.Fields{
			"collector":       j,
			"collection path": bomRoot,
		}).Debugf("can't collect BOMs in multi module mode (gradle), using single module BOM")

		return singleModeBom, nil
	}
	// We only generate one sbom here
	mergedSBOMparam := bomtools.MergeSBOMParam{
		SBOMs: []*cdx.BOM{singleModeBom, multiModeBom},
	}
	return bomtools.MergeSBOMs(mergedSBOMparam)
}

// BootstrapLanguageFiles implements LanguageCollector interface
// BootstrapLanguageFiles implements LanguageCollector interface
func (j JVM) BootstrapLanguageFiles(ctx context.Context, bomRoots []string) []string {
	const bootstrapCmd = "./gradlew"
	const cleanupCmd = "./gradlew --status"

	//for dir, files := range SplitPaths(bomRoots) {
	//	for _, f := range files {
	//		if f == "gradlew" {
	//			log.WithFields(log.Fields{
	//				"collector":       j,
	//				"collection path": dir,
	//			}).Debug("priming gradle cache")
	//
	//			if err := j.executor.shellOut(ctx, dir, bootstrapCmd); err != nil {
	//				log.WithFields(log.Fields{
	//					"error":           err,
	//					"collector":       j,
	//					"collection path": dir,
	//				}).Debug("can't prime gradle cache")
	//				continue
	//			}
	//			log.WithFields(log.Fields{
	//				"collector":       j,
	//				"collection path": dir,
	//			}).Debug("gradle cache primed successfully")
	//
	//			// Add cleanup - stop Gradle daemon
	//			log.WithFields(log.Fields{
	//				"collector":       j,
	//				"collection path": dir,
	//			}).Debug("stopping gradle daemon")
	//			log.Warnf("attempting to cleanup cmd gradle")
	//			if err := j.executor.shellOut(ctx, dir, cleanupCmd); err != nil {
	//				log.WithFields(log.Fields{
	//					"error":           err,
	//					"collector":       j,
	//					"collection path": dir,
	//				}).Debug("failed to stop gradle daemon")
	//			}
	//		}
	//	}
	//}

	return SquashToDirs(bomRoots)
}
