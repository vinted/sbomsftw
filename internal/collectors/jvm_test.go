package collectors

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJVMMatchPredicate(t *testing.T) {
	bundler := JVM{}
	assert.True(t, bundler.matchPredicate(false, "pom.xml"))
	assert.True(t, bundler.matchPredicate(false, "sbt"))
	assert.True(t, bundler.matchPredicate(false, "build.sbt"))
	assert.True(t, bundler.matchPredicate(false, "build.gradle"))
	assert.True(t, bundler.matchPredicate(false, "build.gradle.kts"))

	assert.False(t, bundler.matchPredicate(false, "p0m.xml"))
	assert.False(t, bundler.matchPredicate(true, "build.gradle"))
	assert.False(t, bundler.matchPredicate(true, "pom.xml"))
}

func TestJVMString(t *testing.T) {
	assert.Equal(t, "JVM - (Java/Kotlin/Scala/Groovy)", JVM{}.String())
}
