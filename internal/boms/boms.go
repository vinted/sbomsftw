package boms

import (
	"bytes"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	bootstrapFailedErr = "can't boostrap %s: %w" //Used whenever bundler install fails
)

func FilterOutByScope(bom *cdx.BOM, scope cdx.Scope) *cdx.BOM {
	if bom == nil || bom.Components == nil || len(*bom.Components) == 0 {
		return bom
	}
	//Filter out each component that matches the supplied scope
	var requiredComponents []cdx.Component
	for _, c := range *bom.Components {
		if c.Scope == scope {
			continue
		}
		requiredComponents = append(requiredComponents, c)
	}

	bom.Components = &requiredComponents
	return bom
}

type BOMRootMatcher func(bool, string) bool

type NoRootsFoundError string

func (e NoRootsFoundError) Error() string {
	return string(e)
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
		if d.IsDir() && filepath.Base(path) == ".git" { //todo test this case
			return fs.SkipDir // Don't even traverse to .git directories
		}
		if predicate(d.IsDir(), path) {
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

type BOMType int

const (
	JSON BOMType = iota
	XML
)

func (d BOMType) String() string {
	switch d {
	case JSON:
		return "JSON"
	case XML:
		return "XML"
	default:
		return "Unknown type: " + strconv.Itoa(int(d))
	}
}

type BadBOMTypeError struct{ BOMType BOMType }

func (e BadBOMTypeError) Error() string {
	return fmt.Sprintf("unknown bom type %d", e.BOMType)
}

func BomStringToCDX(format BOMType, bom string) (*cdx.BOM, error) {
	var decoder cdx.BOMDecoder
	switch format {
	case XML:
		decoder = cdx.NewBOMDecoder(strings.NewReader(bom), cdx.BOMFileFormatXML)
	case JSON:
		decoder = cdx.NewBOMDecoder(strings.NewReader(bom), cdx.BOMFileFormatJSON)
	default:
		return nil, BadBOMTypeError{BOMType: format}
	}
	cdxBOM := new(cdx.BOM)
	if err := decoder.Decode(cdxBOM); err != nil {
		return nil, fmt.Errorf("unable to decode string to cdx.BOM: %w", err)
	}
	return cdxBOM, nil
}

func CdxToBOMString(format BOMType, cdxBOM *cdx.BOM) (string, error) {
	result := &bytes.Buffer{}
	var encoder cdx.BOMEncoder
	switch format {
	case XML:
		encoder = cdx.NewBOMEncoder(result, cdx.BOMFileFormatXML)
	case JSON:
		encoder = cdx.NewBOMEncoder(result, cdx.BOMFileFormatJSON)
	default:
		return "", BadBOMTypeError{BOMType: format}
	}
	encoder.SetPretty(true)
	if err := encoder.Encode(cdxBOM); err != nil {
		return "", fmt.Errorf("unable to encode cdx.BOM to string: %w", err)
	}
	return result.String(), nil
}

func ConvertBetweenTypes(inFormat BOMType, outFormat BOMType, bom string) (string, error) {
	cdxBOM, err := BomStringToCDX(inFormat, bom)
	if err != nil {
		return "", fmt.Errorf("can't convert bom from %s to %s: %w", inFormat, outFormat, err)
	}
	return CdxToBOMString(outFormat, cdxBOM)
}

func squashRoots(bomRoots []string) []string {
	var dirsToFiles = make(map[string][]string)
	for _, r := range bomRoots {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}
	squashed := make([]string, 0, len(dirsToFiles))
	for dir, _ := range dirsToFiles {
		squashed = append(squashed, dir)
	}
	return squashed
}

//TODO Add docs
func dirsToFiles(bomRoots []string) map[string][]string {
	var dirsToFiles = make(map[string][]string)
	for _, r := range bomRoots {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}
	return dirsToFiles
}
