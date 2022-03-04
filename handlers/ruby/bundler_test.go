package ruby

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSupportsFile(t *testing.T) {
	bundler := Bundler{}
	assert.True(t, bundler.SupportsFile(Gemfile))
	assert.True(t, bundler.SupportsFile(GemfileLock))
	assert.False(t, bundler.SupportsFile("/etc/passwd"))
}
