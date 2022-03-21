package collectors

/*
collectors package provides implementation of various package managers that
are able to generate SBOMs from a give file system path
*/

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
)

type BOMCollectionFailed string

func (e BOMCollectionFailed) Error() string {
	return string(e)
}

const errUnsupportedRepo = BOMCollectionFailed("BOM collection failed for every root. Unsupported repository")

type BOMCollector interface {
	fmt.Stringer
	matchPredicate(bool, string) bool
	CollectBOM(string) (*cdx.BOM, error)
}
