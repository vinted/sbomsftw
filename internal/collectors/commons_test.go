package collectors

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"regexp"
	"testing"
)

const unableToCreateTempFileErr = "unable to create a lockfile for testing: %s"

type mockCLIExecutor struct{ mock.Mock }

func (m *mockCLIExecutor) bootstrap(bomRoot string, bootstrapCmd string) error {
	args := m.Called(bomRoot, bootstrapCmd)
	return args.Error(0)
}

func (m *mockCLIExecutor) executeCDXGen(bomRoot, shellCMD string) (string, error) {
	args := m.Called(bomRoot, shellCMD)
	return args.String(0), args.Error(1)
}

func createTempDir(t *testing.T) string {
	tempDirName, err := ioutil.TempDir("/tmp", "sa")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing: %s", err)
	}
	return tempDirName
}

func assertGeneratedBOM(t *testing.T, got, expectedFile string) {
	t.Helper()
	expectedBOM, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("unable to read %s file: %s", expectedFile, err)
	}

	//Patch yarn result - BOM uuids are random
	re := regexp.MustCompile("uuid:\\w+-\\w+-\\w+-\\w+-\\w+")
	patchedBOM := re.ReplaceAllString(string(expectedBOM), re.FindString(got))

	//Patch yarn result - BOM timestamps will always differ
	re = regexp.MustCompile(`"timestamp": "\d+-\d+-\d+T\d+:\d+:\d+.\d+Z"`)
	patchedBOM = re.ReplaceAllString(patchedBOM, re.FindString(got))

	//Patch yarn result - BOM external reference path will always differ
	re = regexp.MustCompile(`"url": ".*/tmp/sa\d+/.+"`)
	patchedBOM = re.ReplaceAllString(patchedBOM, re.FindString(got))

	assert.Equal(t, patchedBOM, got)
}
