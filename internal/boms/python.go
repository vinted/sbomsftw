package boms

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"io/ioutil"
	"os"
	fp "path/filepath"
	"regexp"
	"sort"
	"strings"
)

var condaEnvPattern = regexp.MustCompile(`environment.*\.ya?ml`)
var condaDependencyPattern = regexp.MustCompile(`- .*=\d.*`)
var condaLooseDependencyPattern = regexp.MustCompile(`^[\w-]*=\d.*$`)
var supportedPythonFiles = []string{"setup.py", "requirements.txt", "Pipfile.lock", "poetry.lock"}

type Python struct{ executor BOMBridge }

func NewPythonCollector() Python {
	return Python{executor: defaultBOMBridge{}}
}

//matchPredicate implements BOMCollector interface
func (p Python) matchPredicate(isDir bool, filepath string) bool {
	if isDir {
		return false
	}
	filename := fp.Base(filepath)
	for _, f := range supportedPythonFiles {
		if filename == f {
			return true
		}
	}
	if filename == "environment-dev.yml" || filename == "environment-dev.yaml" {
		return false
	}
	//Match conda env files
	return condaEnvPattern.MatchString(filename)
}

func (p Python) generateBOM(bomRoot string) (*cdx.BOM, error) {
	return p.executor.bomFromCdxgen(bomRoot, python)
}

//bootstrap implements BOMCollector interface
func (p Python) bootstrap(bomRoots []string) []string {
	p.conda2Requirements(bomRoots)
	return squashRoots(bomRoots)
}

//String implements BOMCollector interface
func (p Python) String() string {
	return "Python collector"
}

/*
conda2Requirements traverses bom roots and converts all conda environment.yml files to a single requirements.txt file.
This is needed because cdxgen doesn't support conda package manager
*/
func (p Python) conda2Requirements(bomRoots []string) {
	//Extract dependencies from conda environment.yml files
	dependenciesFromCondaEnv := func(condaEnv string) (requirements []string) {
		for _, dependency := range condaDependencyPattern.FindAllString(condaEnv, -1) {
			dependency = strings.TrimPrefix(dependency, "- ")
			if condaLooseDependencyPattern.MatchString(dependency) && len(strings.Split(dependency, "=")) == 2 {
				//CDXGen wants all dependencies in requirements.txt to be with double == signs.
				dependency = strings.Join(strings.Split(dependency, "="), "==")
			}
			requirements = append(requirements, dependency)
		}
		return requirements
	}

	//Filter out duplicates
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
		err := os.WriteFile(fp.Join(dir, "requirements.txt"), []byte(formatted), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "python collector: can't write to requirements file %s", err)
		}
	}

	for dir, files := range dirsToFiles(bomRoots) {
		var requirements []string
		for _, f := range files {
			if condaEnvPattern.MatchString(f) {
				condaEnv, err := ioutil.ReadFile(fp.Join(dir, f))
				if err != nil {
					fmt.Fprintf(os.Stderr, "python collector: can't open conda environment file %s", err)
					continue
				}
				requirements = append(requirements, dependenciesFromCondaEnv(string(condaEnv))...)
			}
		}
		if len(requirements) == 0 {
			continue
		}
		requirements = uniqueRequirements(requirements) //Filter duplicates & convert to set
		currentContents, err := os.ReadFile(fp.Join(dir, "requirements.txt"))
		if err != nil {
			//requirements.txt don't exist. Write to it directly
			writeRequirementsFile(dir, requirements)
			continue
		}
		//requirements.txt exist. Merge conda env and requirements.txt contents
		requirements = uniqueRequirements(append(requirements, strings.Fields(string(currentContents))...))
		writeRequirementsFile(dir, requirements)
	}
}
