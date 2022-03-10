package handlers_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/handlers"
	"testing"
)

func TestSupportsFile(t *testing.T) {
	bundler := handlers.Bundler{}
	assert.True(t, bundler.MatchFile("Gemfile"))
	assert.True(t, bundler.MatchFile("Gemfile.lock"))
	assert.False(t, bundler.MatchFile("/etc/passwd"))
}
