package boms

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"sort"
)

type NoBOMsToMergeError string

func (e NoBOMsToMergeError) Error() string {
	return string(e)
}

//FilterOutByScope TODO Test when scope is nil
func FilterOutByScope(scope cdx.Scope, format BOMType, bom string) (string, error) {
	cdxBOM, err := BomStringToCDX(format, bom)
	if err != nil {
		return "", fmt.Errorf("can't filter out boms by scope: %w", err)
	}
	//Filter out each component that matches the supplied scope
	var requiredComponents []cdx.Component
	for _, c := range *cdxBOM.Components {
		if c.Scope == scope {
			continue
		}
		requiredComponents = append(requiredComponents, c)
	}

	cdxBOM.Components = &requiredComponents
	return CdxToBOMString(format, cdxBOM)
}

//Merge TODO update docs
//Merge takes in a slice of raw BOM strings and their types (JSON or XML). Note that every
//raw bom string inside the slice must be of the same type. I.e. Don't mix JSON and XML strings
//inside the rawBOMs slice - choose a single type.
//Merge filters out duplicate BOM components and merges multiple-lockfiles BOMs into a single one.
//Returned result is a BOM string in JSON format or an error if something went wrong.
func Merge(format BOMType, boms ...string) (*cdx.BOM, error) {
	if len(boms) == 0 {
		return nil, NoBOMsToMergeError("won't merge an empty slice of BOMs - nothing to do")
	}

	mappedBOMs := make([]*cdx.BOM, 0, len(boms))
	for _, b := range boms { //Map each BOM string into cdx.BOM type
		cdxBOM, err := BomStringToCDX(format, b)
		if err != nil {
			return nil, fmt.Errorf("can't merge boms: %w", err)
		}
		mappedBOMs = append(mappedBOMs, cdxBOM)
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
	finalBOM := *mappedBOMs[0]
	finalBOM.Components = &finalComponents

	return &finalBOM, nil //Return string representation of updated cdx.BOM instance
}
