package collectors

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/boms"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRubyBomCollection(t *testing.T) {

	assertCollectedBOM := func(got *cdx.BOM) {
		actualBOM, err := boms.CdxToBOMString(boms.JSON, got)
		assert.NoError(t, err)
		assertGeneratedBOM(t, actualBOM, "synthetic/bundler/bundler-expected-bom.json")
	}

	setup := func(lockfile string) (*mockCLIExecutor, string) {
		tempDir := createTempDir(t)
		if err := os.WriteFile(filepath.Join(tempDir, lockfile), nil, 0644); err != nil {
			t.Fatalf(unableToCreateTempFileErr, err)
		}
		expectedOutput, err := ioutil.ReadFile("synthetic/bundler/bundler-expected-bom.json")
		if err != nil {
			t.Fatalf("unable to read a test file %s", err)
		}
		executor := new(mockCLIExecutor)
		executor.On("executeCDXGen", tempDir, rubyCDXGenCmd).Return(string(expectedOutput), nil)
		return executor, tempDir
	}

	t.Run("collect BOM from Gemfile.lock correctly", func(t *testing.T) {
		executor, testRepo := setup("Gemfile.lock")
		defer os.RemoveAll(testRepo)

		got, err := Ruby{executor: executor}.CollectBOM(testRepo)

		assert.NoError(t, err)
		assertCollectedBOM(got)

		executor.AssertExpectations(t)
		executor.AssertNumberOfCalls(t, "executeCDXGen", 1)
	})

	t.Run("bootstrap Gemfile to Gemfile.lock & collect BOMs correctly", func(t *testing.T) {
		executor, testRepo := setup("Gemfile")
		defer os.RemoveAll(testRepo)

		executor.On("bootstrap", testRepo, rubyBootstrapCmd).Return(nil)

		got, err := Ruby{executor: executor}.CollectBOM(testRepo)

		assert.NoError(t, err)
		assertCollectedBOM(got)

		executor.AssertExpectations(t)
		executor.AssertNumberOfCalls(t, "bootstrap", 1)
		executor.AssertNumberOfCalls(t, "executeCDXGen", 1)
	})

	t.Run("return errUnsupportedRepo when no BOMs were collected", func(t *testing.T) {
		tempDir := createTempDir(t)
		defer os.RemoveAll(tempDir)

		if err := os.WriteFile(filepath.Join(tempDir, "Gemfile.lock"), nil, 0644); err != nil {
			t.Fatalf(unableToCreateTempFileErr, err)
		}

		executor := new(mockCLIExecutor)
		executor.On("executeCDXGen", tempDir, rubyCDXGenCmd).Return("", io.EOF)
		got, err := Ruby{executor: executor}.CollectBOM(tempDir)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, errUnsupportedRepo)
		executor.AssertNumberOfCalls(t, "executeCDXGen", 1)
	})
}

func TestBundlerMatchPredicate(t *testing.T) {
	bundler := Ruby{}
	assert.True(t, bundler.matchPredicate(false, "Gemfile"))
	assert.True(t, bundler.matchPredicate(false, "Gemfile.lock"))
	assert.False(t, bundler.matchPredicate(false, "/etc/passwd"))
	assert.False(t, bundler.matchPredicate(true, "Gemfile"))
}

func TestBundlerString(t *testing.T) {
	assert.Equal(t, "Ruby-Bundler", Ruby{}.String())
}
