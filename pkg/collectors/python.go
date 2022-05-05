package collectors

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	fp "path/filepath"
	"regexp"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

var (
	condaEnvPattern             = regexp.MustCompile(`environment.*\.ya?ml`)
	condaDependencyPattern      = regexp.MustCompile(`- .*=\d.*`)
	condaLooseDependencyPattern = regexp.MustCompile(`^[\w-]*=\d.*$`)
)

type Python struct {
	executor shellExecutor
}

func NewPythonCollector(ctx context.Context) Python {
	return Python{
		executor: newDefaultShellExecutor(ctx),
	}
}

// MatchLanguageFiles implements LanguageCollector interface
func (p Python) MatchLanguageFiles(isDir bool, filepath string) bool {
	if isDir {
		return false
	}
	filename := fp.Base(filepath)

	for _, f := range []string{"setup.py", "requirements.txt", "Pipfile.lock", "poetry.lock"} {
		if filename == f {
			return true
		}
	}

	if filename == "environment-dev.yml" || filename == "environment-dev.yaml" {
		return false
	}
	// Match conda env files
	return condaEnvPattern.MatchString(filename)
}

// GenerateBOM implements LanguageCollector interface
func (p Python) GenerateBOM(bomRoot string) (*cdx.BOM, error) {
	defer func() {
		if err := os.RemoveAll(bomRoot); err != nil {
			log.WithFields(log.Fields{
				"collector": p,
				"error":     err,
			}).Debugf("GenerateBOM: can't remove %s", bomRoot)
		}
	}()
	const language = "python"

	return p.executor.bomFromCdxgen(fp.Dir(bomRoot), language, false)
}

/*BootstrapLanguageFiles implements LanguageCollector interface. Traverses bom roots and converts
all conda environment.yml files to a single requirements.txt file. This is needed because cdxgen
doesn't support conda package manager.
*/
func (p Python) BootstrapLanguageFiles(bomRoots []string) []string {
	// Extract dependencies from conda environment.yml files
	dependenciesFromCondaEnv := func(condaEnv string) (requirements []string) {
		for _, dependency := range condaDependencyPattern.FindAllString(condaEnv, -1) {
			dependency = strings.TrimPrefix(dependency, "- ")
			if condaLooseDependencyPattern.MatchString(dependency) && len(strings.Split(dependency, "=")) == 2 {
				// CDXGen wants all dependencies in requirements.txt to be with double == signs.
				dependency = strings.Join(strings.Split(dependency, "="), "==")
			}
			requirements = append(requirements, dependency)
		}

		return requirements
	}

	// Filter out duplicates
	uniqueRequirements := func(requirements []string) (uniqueRequirements []string) {
		temp := make(map[string]bool)
		for _, r := range requirements {
			if _, ok := temp[r]; !ok {
				temp[r] = true
				uniqueRequirements = append(uniqueRequirements, r)
			}
		}
		sort.Strings(uniqueRequirements)

		return
	}

	writeRequirementsFile := func(dir string, requirements []string) {
		formatted := strings.Join(requirements, "\n")
		if err := os.WriteFile(fp.Join(dir, "requirements.txt"), []byte(formatted), 0o644); err != nil {
			log.WithFields(log.Fields{
				"collector": p,
				"error":     err,
			}).Debug("can't write to requirements file")
		}
	}

	for dir, files := range SplitPaths(bomRoots) {
		var requirements []string
		for _, f := range files {
			if condaEnvPattern.MatchString(f) {
				condaEnv, err := ioutil.ReadFile(fp.Join(dir, f))
				if err != nil {
					log.WithFields(log.Fields{
						"collector": p,
						"error":     err,
					}).Debugf("can't open conda environment file at: %s", fp.Join(dir, f))
					continue
				}
				requirements = append(requirements, dependenciesFromCondaEnv(string(condaEnv))...)
			}
		}
		if len(requirements) == 0 {
			continue
		}
		requirements = uniqueRequirements(requirements) // Filter duplicates & convert to set
		requirementsFilePath := fp.Join(dir, "requirements.txt")
		currentContents, err := os.ReadFile(requirementsFilePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.WithField("collector", p).Debugf("%s: doesn't exist, creating a new one", requirementsFilePath)
			} else {
				log.WithFields(log.Fields{
					"collector": p,
					"error":     err,
				}).Debugf("can't read %s: Creating requirements.txt nonetheless", requirementsFilePath)
			}
			writeRequirementsFile(dir, requirements)
			continue
		}
		// requirements.txt exist. Merge conda env and requirements.txt contents
		requirements = uniqueRequirements(append(requirements, strings.Fields(string(currentContents))...))
		writeRequirementsFile(dir, requirements)
	}
	return bomRoots
}

// String implements LanguageCollector interface
func (p Python) String() string {
	return "python collector"
}
