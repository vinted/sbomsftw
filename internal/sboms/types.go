package sboms

import "fmt"

type BOMResult struct {
	BOM   string
	Error error
}

type BOMGenerator interface {
	fmt.Stringer
	MatchPredicate(string) bool
	GenerateBOM([]string) (string, error)
}
