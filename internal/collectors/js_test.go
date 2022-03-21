package collectors

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/boms"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestJSBomCollection(t *testing.T) {

	assertCollectedBOM := func(got *cdx.BOM) {
		actualBOM, err := boms.CdxToBOMString(boms.JSON, got)
		assert.NoError(t, err)
		assertGeneratedBOM(t, actualBOM, "synthetic/js/js-expected-bom.json")
	}

	setup := func(tempDir string, lockfiles ...string) (*mockCLIExecutor, string) {

		analysisDir, err := ioutil.TempDir(tempDir, "analysis")
		if err != nil {
			t.Fatalf("unable to create temp directory for testing: %s", err)
		}

		for _, l := range lockfiles {
			//Write dummy lockfile
			if err := os.WriteFile(filepath.Join(analysisDir, filepath.Base(l)), nil, 0644); err != nil {
				t.Fatalf(unableToCreateTempFileErr, err)
			}
		}

		expectedOutput, err := ioutil.ReadFile("synthetic/js/js-expected-bom.json")
		if err != nil {
			t.Fatalf("unable to read a test file %s", err)
		}

		executor := new(mockCLIExecutor)
		executor.On("executeCDXGen", analysisDir, jsCDXGenCmd).Return(string(expectedOutput), nil)
		return executor, analysisDir
	}

	t.Run("collect BOMs from lockfiles correctly", func(t *testing.T) {
		testCases := []struct{ lockfiles []string }{
			{lockfiles: []string{"yarn.lock"}},
			{lockfiles: []string{"pnpm-lock.yaml"}},
			{lockfiles: []string{"package-lock.json"}},
			{lockfiles: []string{"package-lock.json", "pnpm-lock.yaml", "yarn.lock"}},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%v must produce correct BOM", tc.lockfiles), func(t *testing.T) {
				tempDir := createTempDir(t)
				defer os.RemoveAll(tempDir)

				executor, testRepo := setup(tempDir, tc.lockfiles...)
				got, err := JS{executor: executor}.CollectBOM(testRepo)

				assert.NoError(t, err)
				assertCollectedBOM(got)

				executor.AssertExpectations(t)
				executor.AssertNumberOfCalls(t, "executeCDXGen", 1)
			})
		}
	})

	t.Run("bootstrap package.json to package-lock.json & collect BOMs correctly", func(t *testing.T) {
		tempDir := createTempDir(t)
		defer os.RemoveAll(tempDir)

		executor, testRepo := setup(tempDir, "package.json")
		executor.On("bootstrap", testRepo, jsBootstrapCmd).Return(nil)

		got, err := JS{executor: executor}.CollectBOM(testRepo)

		assert.NoError(t, err)
		assertCollectedBOM(got)

		executor.AssertExpectations(t)
		executor.AssertNumberOfCalls(t, "bootstrap", 1)
		executor.AssertNumberOfCalls(t, "executeCDXGen", 1)
	})

	t.Run("return errUnsupportedRepo when no BOMs were collected", func(t *testing.T) {
		tempDir := createTempDir(t)
		defer os.RemoveAll(tempDir)

		if err := os.WriteFile(filepath.Join(tempDir, "yarn.lock"), nil, 0644); err != nil {
			t.Fatalf(unableToCreateTempFileErr, err)
		}

		executor := new(mockCLIExecutor)
		executor.On("executeCDXGen", tempDir, jsCDXGenCmd).Return("", io.EOF)

		got, err := JS{executor: executor}.CollectBOM(tempDir)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, errUnsupportedRepo)

		executor.AssertExpectations(t)
		executor.AssertNumberOfCalls(t, "executeCDXGen", 1)
	})
}

func TestJSMatchPredicate(t *testing.T) {

	js := JS{}
	for _, f := range []string{"yarn.lock", "bower.json", "package.json", "pnpm-lock.yaml", "package-lock.json"} {
		assert.True(t, js.matchPredicate(false, f))
	}
	assert.False(t, js.matchPredicate(false, "/etc/passwd"))
	assert.False(t, js.matchPredicate(false, "/tmp/repo/node_modules/yarn.lock"))

	//Special case
	assert.True(t, js.matchPredicate(true, "/tmp/repo/node_modules"))
}

func TestJSString(t *testing.T) {
	assert.Equal(t, "JS/TS-JS", JS{}.String())
}
