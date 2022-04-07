package boms

import (
	"bytes"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	bootstrapFailedErr = "can't boostrap %s: %w" //Used whenever bundler install fails
)

type UnableToMergeBOMsError string

func (e UnableToMergeBOMsError) Error() string {
	return string(e)
}

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

func normalizePURLs(bom *cdx.BOM) *cdx.BOM {
	if bom.Components == nil || len(*bom.Components) == 0 {
		return bom
	}
	var normalized []cdx.Component
	re := regexp.MustCompile(`pkg:\w+/%40[-.\w]+/`)
	for _, c := range *bom.Components {
		var normalizedPURL = c.PackageURL //TODO Add a test for this
		if re.MatchString(normalizedPURL) {
			normalizedPURL = strings.Replace(normalizedPURL, "%40", "", 1)
		}
		wrapped, err := url.Parse(normalizedPURL)
		if err != nil {
			c.PackageURL = normalizedPURL
			normalized = append(normalized, c)
			continue
		}
		c.PackageURL = wrapped.Scheme + ":" + wrapped.Opaque
		normalized = append(normalized, c)
	}
	bom.Components = &normalized
	return bom
}

func normalizePackageNames(bom *cdx.BOM) *cdx.BOM {
	if bom.Components == nil || len(*bom.Components) == 0 {
		return bom
	}
	var normalized []cdx.Component
	for _, c := range *bom.Components {
		var normalizedName = c.Name
		if strings.HasPrefix(c.Name, "@") {
			normalizedName = strings.TrimPrefix(c.Name, "@")
		}
		c.Name = normalizedName
		normalized = append(normalized, c)
	}
	bom.Components = &normalized
	return bom

}

//Merge TODO update docs
//Merge takes in a slice of raw BOM strings and their types (JSON or XML). Note that every
//raw bom string inside the slice must be of the same type. I.e. Don't mix JSON and XML strings
//inside the rawBOMs slice - choose a single type.
//Merge filters out duplicate BOM components and merges multiple-lockfiles BOMs into a single one.
//Returned result is a BOM string in JSON format or an error if something went wrong.
func Merge(boms ...*cdx.BOM) (*cdx.BOM, error) {
	if len(boms) == 0 {
		return nil, UnableToMergeBOMsError("can't merge BOMs - empty list of BOMs supplied")
	}
	for _, b := range boms {
		if b == nil {
			return nil, UnableToMergeBOMsError("can't merge BOMs - BOM list can't contain elements")
		}
	}

	//Gather components from every single cdx.BOM instance
	var allComponents []cdx.Component
	for _, b := range boms {
		b = normalizePackageNames(normalizePURLs(b))
		if b.Components != nil {
			//TODO Add a test for this case as well
			allComponents = append(allComponents, *b.Components...)
		}
	}

	/*
		Filter only unique components - equality determined by purl.
		Also merge in licenses from multiple components into one. This enriches results
	*/
	var uniqComponents = make(map[string]cdx.Component)
	for _, currentComponent := range allComponents {
		previousComponent, ok := uniqComponents[currentComponent.PackageURL]
		if !ok {
			uniqComponents[currentComponent.PackageURL] = currentComponent
			continue
		}
		//If there is licensing info from other SBOMs for the same component - merge those licenses in.
		licensesWereMissing := previousComponent.Licenses == nil || len(*previousComponent.Licenses) == 0
		if licensesWereMissing && currentComponent.Licenses != nil {
			uniqComponents[currentComponent.PackageURL] = currentComponent
		}
	}

	//Make the map iteration stable and in alphabetical order
	keys := make([]string, 0, len(uniqComponents))
	for k := range uniqComponents {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	//Reorder unique components
	finalComponents := make([]cdx.Component, 0, len(keys))
	for _, k := range keys {
		finalComponents = append(finalComponents, uniqComponents[k])
	}

	//Update components of the first element (which is a root project) with all the uniquely merged components
	finalBOM := *boms[0]
	finalBOM.Components = &finalComponents

	//TODO Add a merge for dependencies
	return &finalBOM, nil //Return string representation of updated cdx.BOM instance
}

func attachCPEs(bom *cdx.BOM) *cdx.BOM {

	cpeFromComponent := func(c cdx.Component) string {

		cpeSanitize := func(s string) string {
			return strings.Replace(s, ":", "", -1)
		}

		template := "cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*"
		if c.Group != "" {
			return fmt.Sprintf(template, cpeSanitize(c.Group), cpeSanitize(c.Name), cpeSanitize(c.Version))
		}
		if c.Author != "" {
			return fmt.Sprintf(template, cpeSanitize(c.Author), cpeSanitize(c.Name), cpeSanitize(c.Version))
		}
		return fmt.Sprintf(template, cpeSanitize(c.Name), cpeSanitize(c.Name), cpeSanitize(c.Version))
	}

	finalComponents := make([]cdx.Component, 0, len(*bom.Components))
	for _, c := range *bom.Components {
		c.CPE = cpeFromComponent(c)
		finalComponents = append(finalComponents, c)
	}
	bom.Components = &finalComponents
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
