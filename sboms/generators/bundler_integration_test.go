package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/sboms"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"
)

var tempDirName = "/tmp/" + strconv.FormatInt(time.Now().Unix(), 10) + "/"

func TestBundlerBomGeneration(t *testing.T) {
	setup(t)
	defer teardown(t)

	bundler := Bundler{}
	relativeRoots, err := sboms.FindRoots(os.DirFS(tempDirName), bundler.MatchPredicate)
	if err != nil {
		t.Fatalf("unable to collect BOM roots: %s", err)
	}

	absoluteRoots := sboms.RelativeToAbsoluteRoots(relativeRoots, tempDirName)
	got, err := bundler.GenerateBOM(absoluteRoots)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(got))
	assertGeneratedBOM(t, got[0])
}

func assertGeneratedBOM(t *testing.T, got string) {
	t.Helper()
	expectedBOM, err := ioutil.ReadFile("integration-test-data/bundler_expected_bom.xml")
	if err != nil {
		t.Fatalf("unable to read expected BOM file: %s", err)
	}

	//Patch expected result with since BOM uuids are random
	re := regexp.MustCompile("uuid:\\w+-\\w+-\\w+-\\w+-\\w+")
	patchedBOM := re.ReplaceAllString(string(expectedBOM), re.FindString(got))

	assert.Equal(t, patchedBOM, got)
}

func setup(t *testing.T) {
	const gemfileContents = `
source 'https://rubygems.org'

gem "rspec"    , '~> 3.9.0'
gem "rake"     , '~> 12.3', '>= 12.3.3'
gem "authorizenet"  , '~> 1.9.7'
`
	t.Helper()
	//Set GEM_HOME so that `bundler install` won't ask for sudo password
	err := os.Setenv("GEM_HOME", filepath.Join(os.Getenv("HOME"), ".gem"))
	if err != nil {
		t.Fatal("Unable to set GEM_HOME env variable")
	}

	//Create a temp repository with only a Gemfile in it
	if err := os.Mkdir(tempDirName, 0755); err != nil {
		t.Fatalf("unable to create temp directory for testing: %s", err)
	}
	if err := os.WriteFile(tempDirName+gemfile, []byte(gemfileContents), 0644); err != nil {
		t.Fatalf("unable to write a temp Gemfile for testing: %s", err)
	}
}

func teardown(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(tempDirName); err != nil {
		t.Fatalf("unable to remove temp directory created for testing: %s", err)
	}
}
