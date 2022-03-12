package sboms

import "fmt"

type BOM map[string]string

type BOMGenerator interface {
	fmt.Stringer
	MatchPredicate(string) bool
	GenerateBOM([]string) (string, error)
}
