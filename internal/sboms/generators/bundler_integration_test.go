package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/sboms"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestBOMGeneration(t *testing.T) {
	tempDirName, teardown := setup(t)
	defer teardown()

	bundler := Bundler{}
	relativeRoots, err := sboms.FindRoots(os.DirFS(tempDirName), bundler.MatchPredicate)
	if err != nil {
		t.Fatalf("unable to collect BOM roots: %s", err)
	}

	absoluteRoots := sboms.RelativeToAbsoluteRoots(relativeRoots, tempDirName)
	got, err := bundler.GenerateBOM(absoluteRoots)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(got))
	assertGeneratedBOM(t, got[0], "integration-test-data/bundler_expected_bom.xml")
	assertGeneratedBOM(t, got[1], "integration-test-data/bundler_legacy_expected_bom.xml")
}

/*
setup function creates the following temp repository structure:
/tmp/random-path/Gemfile <- First repo
/tmp/random-path/nitro/Gemfile  <- Second (nested) repo
/tmp/random-path/nitro/jenkins-nitro.gemspec  <- Second (nested) repo
/tmp/random-path/nitro/lib/jenkins_nitro/version.rb  <- Second (nested) repo
*/
func setup(t *testing.T) (tempDirName string, teardown func()) {

	const errMsgTemplate = "unable to create %s for testing: %s"
	tempDirName, err := ioutil.TempDir("/tmp", "sa")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing: %s", err)
	}

	/*
		Create a temp repository with a Gemfile in it - /tmp/unix-timestamp/Gemfile
	*/
	createTopLevelRepository := func() {
		const gemfileContents = `
source 'https://rubygems.org'

gem "rspec"    , '~> 3.9.0'
gem "rake"     , '~> 12.3', '>= 12.3.3'
gem "authorizenet"  , '~> 1.9.7'
`
		gemfilePath := filepath.Join(tempDirName, gemfile)
		if err := os.WriteFile(filepath.Join(tempDirName, gemfile), []byte(gemfileContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, gemfilePath, err)
		}
	}

	/*
		Create a nested temp repository with the following data structure
		/tmp/unix-timestamp/nitro/Gemfile
		/tmp/unix-timestamp/nitro/jenkins-nitro.gemspec
		/tmp/unix-timestamp/nitro/lib/jenkins_nitro/version.rb
	*/

	createNestedRepository := func() {
		const gemfileContents = `
source 'https://rubygems.org'

gemspec
`
		const gemspecContents = `
# coding: utf-8
lib = File.expand_path('../lib', __FILE__)
$LOAD_PATH.unshift(lib) unless $LOAD_PATH.include?(lib)
require 'jenkins_nitro/version'

Gem::Specification.new do |spec|
  spec.name          = "jenkins-nitro"
  spec.version       = JenkinsNitro::VERSION
  spec.authors       = ["John Doe"]
  spec.email         = ["johndoe@acme-corp.com"]
  spec.description   = %q{Command line tool for analyzing Jenkins test duration changes between \
fast and slow builds and pinpointing the cause of slowdown.}
  spec.summary       = %q{Jenkins test suite slowdown analyzer}
  spec.homepage      = "https://github.com/vinted/jenkins-nitro"
  spec.license       = "MIT"

  spec.files         = 'git ls-files'.split($/)
  spec.executables   = spec.files.grep(%r{^bin/}) { |f| File.basename(f) }
  spec.test_files    = spec.files.grep(%r{^(test|spec|features)/})
  spec.require_paths = ["lib"]

  spec.add_development_dependency "bundler", "~> 1.3"
  spec.add_development_dependency "rake"
end
`
		const versionRBContents = `
module JenkinsNitro
  VERSION = "0.0.6"
end
`
		nestedRepoPath := filepath.Join(tempDirName, "nitro/")
		if err := os.MkdirAll(filepath.Join(nestedRepoPath, "lib/jenkins_nitro"), 0755); err != nil {
			t.Fatalf("unable to create temp directory for testing: %s", err)
		}

		gemfilePath := filepath.Join(nestedRepoPath, gemfile)
		if err := os.WriteFile(gemfilePath, []byte(gemfileContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, gemfilePath, err)
		}
		gemspecPath := filepath.Join(nestedRepoPath, "jenkins-nitro.gemspec")
		if err := os.WriteFile(gemspecPath, []byte(gemspecContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, gemspecPath, err)
		}
		versionRBPath := filepath.Join(nestedRepoPath, "lib/jenkins_nitro/version.rb")
		if err := os.WriteFile(versionRBPath, []byte(versionRBContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, versionRBPath, err)
		}
	}

	teardown = func() {
		t.Helper()
		if err := os.RemoveAll(tempDirName); err != nil {
			t.Fatalf("unable to remove temp directory created for testing: %s", err)
		}
	}

	t.Helper()

	//Set GEM_HOME so that `bundler install` won't ask for sudo password
	err = os.Setenv("GEM_HOME", filepath.Join(os.Getenv("HOME"), ".gem"))
	if err != nil {
		t.Fatal("Unable to set GEM_HOME env variable")
	}

	createTopLevelRepository()
	createNestedRepository()
	return tempDirName, teardown
}

func assertGeneratedBOM(t *testing.T, got, assertionFile string) {
	t.Helper()
	expectedBOM, err := ioutil.ReadFile(assertionFile)
	if err != nil {
		t.Fatalf("unable to read expected BOM file: %s", err)
	}

	//Patch expected result with since BOM uuids are random
	re := regexp.MustCompile("uuid:\\w+-\\w+-\\w+-\\w+-\\w+")
	patchedBOM := re.ReplaceAllString(string(expectedBOM), re.FindString(got))

	assert.Equal(t, patchedBOM, got)
}
