package collectors

import (
	"context"
	fp "path/filepath"

	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
)

// Supported files by this collector.
const (
	gemfile     = "Gemfile"
	gemfileLock = "Gemfile.lock"
)

type Ruby struct {
	executor shellExecutor
}

func NewRubyCollector() Ruby {
	return Ruby{
		executor: defaultShellExecutor{},
	}
}

// MatchLanguageFiles implements LanguageCollector interface.
func (r Ruby) MatchLanguageFiles(isDir bool, filepath string) bool {
	if isDir { // Return false immediately - bundler only supports Gemfile & Gemfile.lock files.
		return false
	}
	filename := fp.Base(filepath)

	return filename == gemfile || filename == gemfileLock
}

// String implements LanguageCollector interface.
func (r Ruby) String() string {
	return "ruby collector"
}

// BootstrapLanguageFiles implements LanguageCollector interface.
func (r Ruby) BootstrapLanguageFiles(ctx context.Context, bomRoots []string) []string {
	const bootstrapCmd = "bundler install ||  bundler _1.9_ install || bundler _1.17.3_ install"
	bootstrappedRoots := make([]string, 0, len(bomRoots))

	for dir, files := range SplitPaths(bomRoots) {
		if len(files) == 1 && files[0] == gemfile {
			/*
				BootstrapLanguageFiles by running bundler install. This runs three versions of bundler.
				Latest bundler, 1.9 & 1.17.3 bundler, this is needed for compatability reasons
				when working with old ruby projects.
			*/
			f := log.Fields{
				"collector":       r,
				"collection path": dir,
			}
			log.WithFields(f).Info("Bootstrapping language files")

			if err := r.executor.shellOut(ctx, dir, bootstrapCmd); err != nil {

				log.WithFields(log.Fields{
					"collector": r,
					"error":     err,
				}).Debugf("can't bootstrap language files in: %s", dir)

				continue
			}
		}
		bootstrappedRoots = append(bootstrappedRoots, dir)
	}

	return bootstrappedRoots
}

// GenerateBOM implements LanguageCollector interface.
func (r Ruby) GenerateBOM(ctx context.Context, bomRoot string) (*cdx.BOM, error) {
	const language = "ruby"
	return r.executor.bomFromCdxgen(ctx, bomRoot, language, false)
}
