package bomtools

import cdx "github.com/CycloneDX/cyclonedx-go"

func FilterOutByScope(bom *cdx.BOM, scope cdx.Scope) *cdx.BOM {
	if bom == nil || bom.Components == nil || len(*bom.Components) == 0 {
		return bom
	}
	// Filter out each component that matches the supplied scope
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
