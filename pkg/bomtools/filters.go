package bomtools

import (
	"fmt"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

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

// StripCPEsFromComponents Remove CPEs from all SBOM components
func StripCPEsFromComponents(sbom *cdx.BOM) *cdx.BOM {
	if sbom == nil || sbom.Components == nil || len(*sbom.Components) == 0 {
		return sbom
	}

	requiredComponents := make([]cdx.Component, 0, len(*sbom.Components))

	for _, c := range *sbom.Components {
		updatedComponent := c
		updatedComponent.CPE = ""
		requiredComponents = append(requiredComponents, updatedComponent)
	}

	sbom.Components = &requiredComponents

	return sbom
}

// SetCreatedAtProperty Bakes in the SBOM creation date as a CycloneDX property
func SetCreatedAtProperty(sbom *cdx.BOM) *cdx.BOM {
	if sbom == nil {
		return nil
	}

	createdAt := cdx.Property{
		Name:  "createdAt",
		Value: fmt.Sprintf("%d", time.Now().Unix()),
	}

	if sbom.Properties == nil || len(*sbom.Properties) == 0 {
		sbom.Properties = &[]cdx.Property{createdAt}
		return sbom
	}

	// If some properties exist, simply append the 'createdAt' property
	updatedProperties := append(*sbom.Properties, createdAt)
	sbom.Properties = &updatedProperties

	return sbom
}
