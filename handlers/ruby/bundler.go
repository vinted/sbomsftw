package ruby

const Gemfile = "Gemfile"
const GemfileLock = "Gemfile.lock"

type Handler interface {
	SupportsFile()
}

type Bundler struct{}

func (b Bundler) SupportsFile(filepath string) bool {
	return filepath == Gemfile || filepath == GemfileLock
}
