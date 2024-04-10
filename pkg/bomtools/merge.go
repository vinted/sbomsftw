package bomtools

import (
	"errors"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
)

var ErrNoBOMsToMerge = errors.New("merge_boms: can't merge empty list of BOMs")

/*
stripChecksumIfExists utility function to strip trailing checksums.

Sometimes we will encounter a package PURL or CPE in the following format:
pkg:npm/next@11.1.4_6ae8aab56bccab9c135b13f4dcebcfdd
or
cpe:2.3:a:next:next:11.1.4_6ae8aab56bccab9c135b13f4dcebcfdd:*:*:*:*:*:*:*

These checksums must be stripped off, because Dependency Track analyzer will fail to
find vulnerabilities for these packages. Call this function for PURL & CPE strings
*/
func stripChecksumIfExists(candidate string) string {
	checksumRe := regexp.MustCompile(`([-.\w]+)_[0-9a-f]{32}`)

	matches := checksumRe.FindStringSubmatch(candidate)
	if len(matches) >= 2 {
		return strings.Replace(candidate, matches[0], matches[1], 1)
	}

	return candidate
}

func normalizePURLs(bom *cdx.BOM) *cdx.BOM {
	if bom.Components == nil || len(*bom.Components) == 0 {
		return bom
	}

	var normalized []cdx.Component
	encodedPurlRe := regexp.MustCompile(`pkg:\w+/%40[-.\w]+/`)
	versionedPurlRe := regexp.MustCompile(`@v[-.\w]+$`)

	for _, c := range *bom.Components {
		if c.Type == cdx.ComponentTypeApplication {
			// Don't normalize PURLs for application type components
			normalized = append(normalized, c)

			continue
		}
		normalizedPURL := c.PackageURL

		if encodedPurlRe.MatchString(normalizedPURL) {
			normalizedPURL = strings.Replace(normalizedPURL, "%40", "", 1)
		}
		if versionedPurlRe.MatchString(normalizedPURL) {
			normalizedPURL = strings.Replace(normalizedPURL, "@v", "@", 1)
		}

		normalizedPURL = stripChecksumIfExists(normalizedPURL)

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

	normalized := make([]cdx.Component, 0, len(*bom.Components))

	for _, c := range *bom.Components {
		normalizedName := c.Name
		if strings.HasPrefix(c.Name, "@") {
			normalizedName = strings.TrimPrefix(c.Name, "@")
		}
		c.Name = normalizedName
		normalized = append(normalized, c)
	}
	bom.Components = &normalized
	return bom
}

/*
normalizeCPEs: strips trailing checksums from CPE strings
*/
func normalizeCPEs(bom *cdx.BOM) *cdx.BOM {
	if bom.Components == nil || len(*bom.Components) == 0 {
		return bom
	}

	normalized := make([]cdx.Component, 0, len(*bom.Components))

	for _, c := range *bom.Components {
		c.CPE = stripChecksumIfExists(c.CPE)
		normalized = append(normalized, c)
	}
	bom.Components = &normalized
	return bom
}

type element interface {
	cdx.Hash | cdx.Property | cdx.LicenseChoice | cdx.ExternalReference
}

func mergeCollection[T element](src, dst []T) []T {
	contains := func(s []T, elem T) bool {
		for _, e := range s {
			if reflect.DeepEqual(e, elem) {
				return true
			}
		}

		return false
	}

	if len(src) == 0 {
		return dst
	}

	results := make([]T, len(dst))
	copy(results, dst)

	for _, candidate := range src {
		if contains(dst, candidate) {
			continue
		}
		results = append(results, candidate)
	}

	return results
}

func mergeAllByPURL(component *cdx.Component, allComponents []*cdx.Component) *cdx.Component {
	if component.Type != cdx.ComponentTypeLibrary {
		// Only merge library components - other components don't guarantee to have a valid Package URL
		return component
	}
	var componentsToMerge []*cdx.Component
	// Filter components to merge by the Package URL given
	for _, c := range allComponents {
		if c.PackageURL == component.PackageURL {
			componentsToMerge = append(componentsToMerge, c)
		}
	}

	hashes := make([]cdx.Hash, 0)
	properties := make([]cdx.Property, 0)
	licenses := make([]cdx.LicenseChoice, 0)
	externalRefs := make([]cdx.ExternalReference, 0)
	mergedComponent := &cdx.Component{ // Create the resulting component
		Hashes:             &hashes,
		Properties:         &properties,
		Licenses:           (*cdx.Licenses)(&licenses),
		ExternalReferences: &externalRefs,
	}

	// Merge everything from multiple components into one
	for _, c := range componentsToMerge {
		mergedComponent.BOMRef = c.PackageURL
		mergedComponent.PackageURL = c.PackageURL
		if c.Type != "" {
			mergedComponent.Type = c.Type
		}
		if c.CPE != "" {
			mergedComponent.CPE = c.CPE
		}
		if c.Name != "" {
			mergedComponent.Name = c.Name
		}
		if c.Version != "" {
			mergedComponent.Version = c.Version
		}
		if c.Description != "" {
			mergedComponent.Description = c.Description
		}
		if c.Scope != "" {
			mergedComponent.Scope = c.Scope
		}
		if c.Hashes != nil {
			h := mergeCollection[cdx.Hash](*c.Hashes, *mergedComponent.Hashes)
			mergedComponent.Hashes = &h
		}
		if c.Properties != nil {
			p := mergeCollection[cdx.Property](*c.Properties, *mergedComponent.Properties)
			mergedComponent.Properties = &p
		}
		if c.Licenses != nil {
			l := mergeCollection[cdx.LicenseChoice](*c.Licenses, *mergedComponent.Licenses)
			mergedComponent.Licenses = (*cdx.Licenses)(&l)
		}
		if c.ExternalReferences != nil {
			e := mergeCollection[cdx.ExternalReference](*c.ExternalReferences, *mergedComponent.ExternalReferences)
			mergedComponent.ExternalReferences = &e
		}
	}
	return mergedComponent
}

func MergeSBOMs(sboms ...*cdx.BOM) (*cdx.BOM, error) {
	// Validate we are working with legit input
	if len(sboms) == 0 {
		return nil, ErrNoBOMsToMerge
	}
	for _, b := range sboms {
		if b == nil {
			return nil, ErrNoBOMsToMerge
		}
	}

	// Gather components from every single cdx.BOM instance
	var allComponents []*cdx.Component
	for _, b := range sboms {
		b = normalizeCPEs(normalizePackageNames(normalizePURLs(b)))
		if b.Components != nil {
			components := *b.Components
			for i := range components {
				allComponents = append(allComponents, &components[i])
			}
		}
	}

	/*
		Filter only unique components - equality determined by purl.
		Also merge in licenses from multiple components into one. This enriches results
	*/
	purlsToComponents := make(map[string]*cdx.Component)
	for _, c := range allComponents {
		purlsToComponents[c.PackageURL] = c
	}
	for purl, c := range purlsToComponents {
		purlsToComponents[purl] = mergeAllByPURL(c, allComponents)
	}

	// Sort components alphabetically by their PURL
	keys := make([]string, 0, len(purlsToComponents))
	for k := range purlsToComponents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedComponents := make([]cdx.Component, 0, len(keys))
	for _, k := range keys {
		sortedComponents = append(sortedComponents, *purlsToComponents[k])
	}

	// create cdxComponent
	components := []cdx.Component{{
		Author:  "vinted",
		Name:    "sa-collector",
		Version: "0.5.0", // TODO Extract somewhere else later on
	},
	}

	// Reconstruct final bom
	bom := cdx.NewBOM()
	bom.Components = &sortedComponents
	bom.SerialNumber = uuid.New().URN()
	bom.Metadata = &cdx.Metadata{
		Timestamp: time.Now().Format(time.RFC3339),
		Component: &components[0],
	}

	return bom, nil
}
