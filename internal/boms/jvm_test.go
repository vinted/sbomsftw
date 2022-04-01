package boms

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJVMMatchPredicate(t *testing.T) {
	jvm := JVM{}
	assert.True(t, jvm.matchPredicate(false, "pom.xml"))
	assert.True(t, jvm.matchPredicate(false, "sbt"))
	assert.True(t, jvm.matchPredicate(false, "build.sbt"))
	assert.True(t, jvm.matchPredicate(false, "build.gradle"))
	assert.True(t, jvm.matchPredicate(false, "build.gradle.kts"))

	assert.False(t, jvm.matchPredicate(false, "p0m.xml"))
	assert.False(t, jvm.matchPredicate(true, "build.gradle"))
	assert.False(t, jvm.matchPredicate(true, "pom.xml"))
}

func TestJVMString(t *testing.T) {
	assert.Equal(t, "JVM - (Java/Kotlin/Scala/Groovy)", JVM{}.String())
}
