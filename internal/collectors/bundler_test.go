package collectors

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchFile(t *testing.T) {
	bundler := Bundler{}
	assert.True(t, bundler.matchPredicate("Gemfile"))
	assert.True(t, bundler.matchPredicate("Gemfile.lock"))
	assert.False(t, bundler.matchPredicate("/etc/passwd"))
}

func TestString(t *testing.T) {
	assert.Equal(t, "Ruby-Bundler", Bundler{}.String())
}
