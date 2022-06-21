package bomtools

import cdx "github.com/CycloneDX/cyclonedx-go"

// FilterOutByScope Filter out SBOM components that don't have the specified scope
func FilterOutByScope(sbom *cdx.BOM, scope cdx.Scope) *cdx.BOM {
	if sbom == nil || sbom.Components == nil || len(*sbom.Components) == 0 {
		return sbom
	}
	// Filter out each component that matches the supplied scope
	requiredComponents := make([]cdx.Component, 0, len(*sbom.Components))

	for _, c := range *sbom.Components {
		if c.Scope == scope {
			continue
		}
		requiredComponents = append(requiredComponents, c)
	}

	sbom.Components = &requiredComponents

	return sbom
}

// FilterOutComponentsWithoutAType Filter SBOM components that have an empty type
func FilterOutComponentsWithoutAType(sbom *cdx.BOM) *cdx.BOM {
	if sbom == nil || sbom.Components == nil || len(*sbom.Components) == 0 {
		return sbom
	}

	requiredComponents := make([]cdx.Component, 0, len(*sbom.Components))

	for _, c := range *sbom.Components {
		if c.Type == "" { // Every component must have a valid type
			continue
		}
		requiredComponents = append(requiredComponents, c)
	}

	sbom.Components = &requiredComponents

	return sbom
}
