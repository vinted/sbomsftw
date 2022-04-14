package collectors

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"regexp"
	"testing"
)

const unableToCreateTempFileErr = "unable to create a lockfile for testing: %s"
const unableToCreateTempDirErr = "unable to create a temp directory for testing: %s"

type mockBOMBridge struct{ mock.Mock }

func (m *mockBOMBridge) shellOut(bomRoot string, shellCmd string) (string, error) {
	args := m.Called(bomRoot, shellCmd)
	return args.String(0), args.Error(1)
}

func (m *mockBOMBridge) bomFromCdxgen(bomRoot, language string) (*cdx.BOM, error) {
	args := m.Called(bomRoot, language)
	return args.Get(0).(*cdx.BOM), args.Error(1)
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
