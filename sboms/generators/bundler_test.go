package generators

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchFile(t *testing.T) {
	bundler := Bundler{}
	assert.True(t, bundler.MatchPredicate(gemfile))
	assert.True(t, bundler.MatchPredicate(gemfileLock))
	assert.False(t, bundler.MatchPredicate("/etc/passwd"))
}

func TestString(t *testing.T) {
	assert.Equal(t, "Ruby-Bundler", Bundler{}.String())
}

func TestNormalizeRoots(t *testing.T) {
	rawRoots := []string{
		"/dir/Gemfile",
		"/dir/Gemfile.lock",
		"/dir/inner-dir/Gemfile",
		"/dir/inner-dir/deep-dir/Gemfile",
		"/dir/inner-dir/deep-dir/Gemfile.lock",
		"/dir/inner-dir/deep-dir/deepest-dir/Gemfile.lock",
	}
	expectedRoots := []string{
		"/dir/Gemfile.lock",
		"/dir/inner-dir/Gemfile",
		"/dir/inner-dir/deep-dir/Gemfile.lock",
		"/dir/inner-dir/deep-dir/deepest-dir/Gemfile.lock",
	}
	assert.ElementsMatch(t, expectedRoots, Bundler{}.normalizeRoots(rawRoots))
}
