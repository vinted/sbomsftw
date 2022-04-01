package boms

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	fp "path/filepath"
	"regexp"
	"strings"
)

type Golang struct{ executor CLIExecutor }

//BOM Generation commands
const (
	goMod                  = "go.mod"
	goSum                  = "go.sum"
	goPkg                  = "Gopkg.lock"
	cdxgenCmd              = "export FETCH_LICENSE=true && cdxgen --type golang"
	cyclonedxGoModTemplate = "cyclonedx-gomod app -json -std=true -licenses -main %s 2>/dev/null"
)

func NewGolangCollector() Golang {
	return Golang{executor: defaultCLIExecutor{}}
}

//matchPredicate implements BOMCollector interface
func (g Golang) matchPredicate(isDir bool, filepath string) bool {
	for _, p := range strings.Split(fp.Dir(filepath), string(os.PathSeparator)) {
		if p == "vendor" { //Ignore files in vendor directory - add a test for this
			return false
		}
	}
	if isDir {
		return false
	}
	filename := fp.Base(filepath)
	return filename == goMod || filename == goSum || filename == goPkg
}

func (g Golang) findEntryPoints(bomRoot string) ([]string, error) {
	var entryPoints []string
	re := regexp.MustCompile("func\\s*main\\s*\\(\\s*\\)") //Grep for entry points in go source
	err := fs.WalkDir(os.DirFS(bomRoot), ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && (path == "vendor" || path == ".git") { //Add a test case for this one
			return fs.SkipDir
		}
		if err != nil {
			return fmt.Errorf("unable to walk file system path: %w", err)
		}
		if d.IsDir() {
			return nil
		}
		data, err := ioutil.ReadFile(fp.Join(bomRoot, path))
		if err != nil {
			fmt.Println(fmt.Errorf("unable to read %s: %w", path, err))
			return nil
		}
		if matches := re.MatchString(string(data)); matches {
			entryPoints = append(entryPoints, fp.Dir(path))
		}
		return nil
	})
	return entryPoints, err
}

func (g Golang) generateBOM(bomRoot string) (string, error) {
	entryPoints, err := g.findEntryPoints(bomRoot)
	if err != nil || len(entryPoints) == 0 {
		fmt.Println("no golang entry points found - falling back to cdxgen")
		return g.executor.executeCDXGen(bomRoot, cdxgenCmd)
	}

	var boms []string
	for _, ep := range entryPoints {
		command := fmt.Sprintf(cyclonedxGoModTemplate, ep)
		result, err := g.executor.shellOut(bomRoot, command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't generate BOMs with cyclonedx-gomod %s - falling back to cdxgen\n", err)
			return g.executor.executeCDXGen(bomRoot, cdxgenCmd)
		}
		boms = append(boms, result)
	}
	unified, err := Merge(JSON, boms...)
	if err != nil {
		return "", err
	}
	return CdxToBOMString(JSON, unified)
}

//bootstrap implements BOMCollector interface
func (g Golang) bootstrap(bomRoots []string) []string {
	return squashRoots(bomRoots)
}

//String implements BOMCollector interface
func (g Golang) String() string {
	return "Golang collector"
}
