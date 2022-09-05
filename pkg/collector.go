package pkg

import (
	"context"
	"fmt"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

/*
Collector interface used by external collectors such as syft or retirejs collector.
Implementations of this interface can be passed to Repository struct to generate a base BOM.
*/
type Collector interface {
	/*
		Stringer Provide a user-friendly description of this collector.
		This description will be present in the output log.
	*/
	fmt.Stringer
	/*
		GenerateBOM given a ctx & filesystem path - construct a BOM or return an error.
		Anyone who implements this interface and passes their implementation to Repository
		struct should be aware that GenerateBOM will be called once per repository and
		the string parameter passed in will be the filesystem root of that repository.
	*/
	GenerateBOM(context.Context, string) (*cdx.BOM, error)
}

/*
LanguageCollector interface used by language specific collectors. Implementations of
this interface can be passed to Repository struct to generate BOM for a specific language.
*/
type LanguageCollector interface {
	/*
		Collector - borrow all stub methods from Collector interface. However, GenerateBOM
		will be called for each directory where language specific files reside in, instead of
		once per repository filesystem root.
	*/
	Collector
	/*
		MatchLanguageFiles provides a way to identify language specific files for collector.
		When LanguageCollector is passed to Repository struct, this method will be called for each file
		and directory inside the filesystem path of a repository. The first parameter indicates whether the
		file node is a directory or not. The second parameter is the filepath.
		E.g. Rust collector implementation should return true for Cargo.lock and Cargo.toml files
	*/
	MatchLanguageFiles(bool, string) bool
	/*
		BootstrapLanguageFiles when implementations of LanguageCollector are passed to Repository struct,
		this method is called with all the language specific files identified by the MatchLanguageFiles method.
		File paths passed to this method are always absolute.
		Sometimes when implementing this method we can return the files paths passed in without any modifications.
		However, in some cases we might need to do some preprocessing.
		E.g. Ruby collector might need to generate lockfiles if only Gemfiles exist by running bundler install.
		When passed to Repository struct, the files returned by this method are then looped over with
		GenerateBOM calls.
	*/
	BootstrapLanguageFiles(context.Context, []string) []string
}
