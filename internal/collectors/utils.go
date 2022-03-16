package collectors

import (
	"bytes"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type BOMType int

const (
	JSON BOMType = iota
	XML
)

type BOMRootMatcher func(string) bool

type NoRootsFoundError string

func (e NoRootsFoundError) Error() string {
	return string(e)
}

type NoBOMsToMergeError string

func (e NoBOMsToMergeError) Error() string {
	return string(e)
}

type BadBOMTypeError struct{ BOMType BOMType }

func (e BadBOMTypeError) Error() string {
	return fmt.Sprintf("unknown bom type %d", e.BOMType)
}

/*
findRoots walks the given file-system path (preferably a git repository) and
collects relative BOM roots using the provided BOMRootMatcher function.
A BOM root is considered a file that can be used by cyclonedx-tools to create SBOMs. E.g.
Gemfile.lock or go.mod. Empty results might be returned if the BOMRootMatcher doesn't support
the repository or if the repository walk fails. Which can happen in some rare cases. E.g.
missing permissions.
*/
func findRoots(fileSystem fs.FS, predicate BOMRootMatcher) ([]string, error) {
	var roots []string
	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("unable to walk file system path: %w", err)
		}
		if basename := filepath.Base(path); predicate(basename) && !d.IsDir() {
			roots = append(roots, path)
		}
		return nil
	})
	return roots, err
}

/*relativeToAbsoluteRoots takes in a slice of relative BOM roots (as returned from findRoots) function
and a parent directory. Returns absolute roots prefixed with parent dir. E.g. given relative roots:
[]string{
		"Packages",
		"Packages.lock",
		"inner-dir/Packages.lock",
		"inner-dir/deepest-dir/Packages.lock",
	}
and a parent directory of '/tmp/test-repository' returns
[]string{
		"/tmp/test-repository/Packages",
		"/tmp/test-repository/Packages.lock",
		"/tmp/test-repository/inner-dir/Packages.lock",
		"/tmp/test-repository/inner-dir/deepest-dir/Packages.lock",
	}
*/
func relativeToAbsoluteRoots(parentDir string, relativeRoots ...string) (absoluteRoots []string) {
	for _, relativeRoot := range relativeRoots {
		absoluteRoots = append(absoluteRoots, filepath.Join(parentDir, relativeRoot))
	}
	return
}

func repoToRoots(repoPath string, predicate BOMRootMatcher) ([]string, error) {
	relativeRoots, err := findRoots(os.DirFS(repoPath), predicate)
	if err != nil {
		return nil, fmt.Errorf("can't to convert repo to roots: %w", err)
	}
	absoluteRoots := relativeToAbsoluteRoots(repoPath, relativeRoots...)
	if len(absoluteRoots) == 0 {
		return nil, NoRootsFoundError("No BOM roots found for supplied predicate")
	}
	return absoluteRoots, nil
}

// NormalizeRoots TODO Add docs
func normalizeRoots(preferredFile string, rawRoots ...string) (result []string) {
	var dirsToFiles = make(map[string][]string)
	for _, r := range rawRoots {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}
	for dir, files := range dirsToFiles {
		if len(files) == 1 {
			//Just return the single file, doesn't matter if it's a lockfile or not
			result = append(result, filepath.Join(dir, files[0]))
			continue
		}
		//Use preferred file (lockfile) whenever multiple BOM roots are present
		result = append(result, filepath.Join(dir, preferredFile))
	}
	return
}

//Merge TODO update docs
//Merge takes in a slice of raw BOM strings and their types (JSON or XML). Note that every
//raw bom string inside the slice must be of the same type. I.e. Don't mix JSON and XML strings
//inside the rawBOMs slice - choose a single type.
//Merge filters out duplicate BOM components and merges multiple BOMs into a single one.
//Returned result is a BOM string in JSON format or an error if something went wrong.
func Merge(format BOMType, boms ...string) (string, error) {
	if len(boms) == 0 {
		return "", NoBOMsToMergeError("won't merge an empty slice of BOMs - nothing to do")
	}

	var mappedBOMs []*cdx.BOM
	for _, b := range boms { //Decode each BOM string into cdx.BOM type
		bom := new(cdx.BOM)

		var decoder cdx.BOMDecoder
		switch format {
		case XML:
			decoder = cdx.NewBOMDecoder(strings.NewReader(b), cdx.BOMFileFormatXML)
		case JSON:
			decoder = cdx.NewBOMDecoder(strings.NewReader(b), cdx.BOMFileFormatJSON)
		default:
			return "", BadBOMTypeError{BOMType: format}
		}
		if err := decoder.Decode(bom); err != nil {
			return "", fmt.Errorf("unable to decode raw string to cdx.BOM: %w", err)
		}
		mappedBOMs = append(mappedBOMs, bom)
	}

	//Gather components from every single cdx.BOM instance
	var allComponents []cdx.Component
	for _, m := range mappedBOMs {
		allComponents = append(allComponents, *m.Components...)
	}

	//Filter only unique components - equality determined by purl
	var uniqComponents = make(map[string]cdx.Component)
	for _, component := range allComponents {
		uniqComponents[component.PackageURL] = component
	}

	//Make the map iteration stable and in alphabetical order
	var keys []string
	for k := range uniqComponents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	//Reorder unique components
	var finalComponents []cdx.Component
	for _, k := range keys {
		finalComponents = append(finalComponents, uniqComponents[k])
	}

	//Update components of the first element (which is a root project) with all the uniquely merged components
	finalBOM := *mappedBOMs[0]
	finalBOM.Components = &finalComponents

	//Convert updated cdx.BOM instance to a nice JSON string
	result := &bytes.Buffer{}
	encoder := cdx.NewBOMEncoder(result, cdx.BOMFileFormatJSON)
	encoder.SetPretty(true)
	if err := encoder.Encode(&finalBOM); err != nil {
		return "", fmt.Errorf("unable to encode cdx.BOM to a JSON string: %w", err)
	}
	return result.String(), nil
}
