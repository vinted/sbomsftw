package bomtools

import (
	"errors"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var ErrNoBOMsToMerge = errors.New("merge_boms: can't merge empty list of BOMs")

func normalizePURLs(bom *cdx.BOM) *cdx.BOM {
	if bom.Components == nil || len(*bom.Components) == 0 {
		return bom
	}
	var normalized []cdx.Component
	re := regexp.MustCompile(`pkg:\w+/%40[-.\w]+/`)
	for _, c := range *bom.Components {
		if c.Type == cdx.ComponentTypeApplication {
			//Don't normalize PURLs for application type components
			normalized = append(normalized, c)
			continue
		}
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
		//Only merge library components - other components don't guarantee to have a valid Package URL
		return component
	}
	var componentsToMerge []*cdx.Component
	//Filter components to merge by the Package URL given
	for _, c := range allComponents {
		if c.PackageURL == component.PackageURL {
			componentsToMerge = append(componentsToMerge, c)
		}
	}

	hashes := make([]cdx.Hash, 0)
	properties := make([]cdx.Property, 0)
	licenses := make([]cdx.LicenseChoice, 0)
	externalRefs := make([]cdx.ExternalReference, 0)
	mergedComponent := &cdx.Component{ //Create the resulting component
		Hashes:             &hashes,
		Properties:         &properties,
		Licenses:           (*cdx.Licenses)(&licenses),
		ExternalReferences: &externalRefs,
	}

	//Merge everything from multiple components into one
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

func MergeBoms(boms ...*cdx.BOM) (*cdx.BOM, error) {
	//Validate we are working with legit input
	if len(boms) == 0 {
		return nil, ErrNoBOMsToMerge
	}
	for _, b := range boms {
		if b == nil {
			return nil, ErrNoBOMsToMerge
		}
	}

	//Gather components from every single cdx.BOM instance
	var allComponents []*cdx.Component
	for _, b := range boms {
		b = normalizePackageNames(normalizePURLs(b))
		if b.Components != nil {
			components := *b.Components
			for i, _ := range components {
				allComponents = append(allComponents, &components[i])
			}
		}
	}

	/*
		Filter only unique components - equality determined by purl.
		Also merge in licenses from multiple components into one. This enriches results
	*/
	var purlsToComponents = make(map[string]*cdx.Component)
	for _, c := range allComponents {
		purlsToComponents[c.PackageURL] = c
	}
	for purl, c := range purlsToComponents {
		purlsToComponents[purl] = mergeAllByPURL(c, allComponents)
	}

	//Sort components alphabetically by their PURL
	keys := make([]string, 0, len(purlsToComponents))
	for k := range purlsToComponents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedComponents := make([]cdx.Component, 0, len(keys))
	for _, k := range keys {
		sortedComponents = append(sortedComponents, *purlsToComponents[k])
	}

	//Reconstruct final bom
	bom := cdx.NewBOM()
	bom.Components = &sortedComponents
	bom.SerialNumber = uuid.New().URN()
	bom.Metadata = &cdx.Metadata{
		Timestamp: time.Now().Format(time.RFC3339),
		Tools: &[]cdx.Tool{
			{
				Vendor:  "vinted",
				Name:    "sa-collector",
				Version: "0.1.0", //TODO Extract somewhere else later on
			},
		},
	}
	//Reattach dependency graph
	for _, b := range boms {
		isDependencyGraphPresent := b.Dependencies != nil && len(*b.Dependencies) > 0
		if isDependencyGraphPresent {
			bom.Dependencies = b.Dependencies
			break
		}
	}
	//Reattach external refs
	for _, b := range boms {
		areExternalRefsPresent := b.ExternalReferences != nil && len(*b.ExternalReferences) > 0
		if areExternalRefsPresent {
			bom.ExternalReferences = b.ExternalReferences
			break
		}
	}
	return bom, nil
}
