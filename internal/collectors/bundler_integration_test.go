package collectors_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/collectors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestBOMGeneration(t *testing.T) {
	t.Run("collect BOMs from multiple ruby repositories correctly", func(t *testing.T) {
		tempDirName, teardown := setup(t)
		defer teardown()

		got, err := collectors.Bundler{}.CollectBOM(tempDirName)
		assert.NoError(t, err)
		assertGeneratedBOM(t, got)
	})

	t.Run("return an error when BOM collection fails", func(t *testing.T) {
		//Setup
		testingDir, err := ioutil.TempDir("/tmp", "sa")
		if err != nil {
			t.Fatalf("unable to create temp directory for testing: %s", err)
		}
		gemfilePath := filepath.Join(testingDir, "Gemfile.lock")
		if err := os.WriteFile(gemfilePath, []byte("👻 Invalid Gemfile.lock contents"), 0644); err != nil {
			t.Fatal("unable to create Gemfile.lock for testing!")
		}

		teardown := func() {
			if err := os.RemoveAll(testingDir); err != nil {
				t.Fatalf("unable to remove temp directory created for testing: %s", err)
			}
		}
		defer teardown()

		got, err := collectors.Bundler{}.CollectBOM(testingDir)
		assert.Empty(t, got)
		assert.ErrorIs(t, err, collectors.BOMCollectionFailed("BOM generation failed for every root"))
	})
}

func assertGeneratedBOM(t *testing.T, got string) {
	t.Helper()
	expectedBOM, err := ioutil.ReadFile("integration-test-data/bundler_expected_bom.json")
	if err != nil {
		t.Fatalf("unable to read expected BOM file: %s", err)
	}

	//Patch expected result - BOM uuids are random
	re := regexp.MustCompile("uuid:\\w+-\\w+-\\w+-\\w+-\\w+")
	patchedBOM := re.ReplaceAllString(string(expectedBOM), re.FindString(got))

	//Patch expected result - BOM timestamps will always differ
	re = regexp.MustCompile(`"timestamp": "\d+-\d+-\d+T\d+:\d+:\d+.\d+Z"`)
	patchedBOM = re.ReplaceAllString(patchedBOM, re.FindString(got))

	//Patch expected result - BOM external reference path will always differ
	re = regexp.MustCompile(`"url": ".*/tmp/sa\d+/Gemfile.lock`)
	patchedBOM = re.ReplaceAllString(patchedBOM, re.FindString(got))

	assert.Equal(t, patchedBOM, got)
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

	//Create a temp repository with a Gemfile in it - /tmp/unix-timestamp/Gemfile
	createTopLevelRepository := func() {
		const gemfileContents = `
source 'https://rubygems.org'

gem "rspec"    , '~> 3.9.0'
gem "rake"     , '~> 12.3', '>= 12.3.3'
gem "authorizenet"  , '~> 1.9.7'
`
		gemfile := filepath.Join(tempDirName, "Gemfile")
		if err := os.WriteFile(gemfile, []byte(gemfileContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, gemfile, err)
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
  spec.homepage      = "https://requests.com/vinted/jenkins-nitro"
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

		gemfile := filepath.Join(nestedRepoPath, "Gemfile")
		if err := os.WriteFile(gemfile, []byte(gemfileContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, gemfile, err)
		}
		gemspec := filepath.Join(nestedRepoPath, "jenkins-nitro.gemspec")
		if err := os.WriteFile(gemspec, []byte(gemspecContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, gemspec, err)
		}
		versionRb := filepath.Join(nestedRepoPath, "lib/jenkins_nitro/version.rb")
		if err := os.WriteFile(versionRb, []byte(versionRBContents), 0644); err != nil {
			t.Fatalf(errMsgTemplate, versionRb, err)
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
