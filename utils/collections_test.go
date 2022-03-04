package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKeys(t *testing.T) {
	key1 := "key-1"
	key2 := "key-2"
	key3 := "key-3"
	got := Keys(map[string]bool{key1: true, key2: true, key3: true})
	assert.ElementsMatch(t, []string{key1, key2, key3}, got)
}
