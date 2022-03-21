package collectors

import "fmt"

type BOMCollectionFailed string

func (e BOMCollectionFailed) Error() string {
	return string(e)
}

type BOMCollector interface {
	fmt.Stringer
	matchPredicate(string) bool
	CollectBOM(string) (string, error)
}
