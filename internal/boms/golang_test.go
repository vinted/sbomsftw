package boms

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

//Entry points used for repository structure
const (
	firstEntryPoint  = "cmd/test-app"
	secondEntryPoint = "cmd/test-app-2"
	// Add a case for vendor
)

func setupGolangTestRepo(t *testing.T) string {
	tempDir := createTempDir(t)

	createChildDir := func(dir string) {
		if err := os.Mkdir(filepath.Join(tempDir, dir), 0777); err != nil {
			t.Fatalf(unableToCreateTempDirErr, err) //extract to const
		}
	}

	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), nil, 0644); err != nil {
		t.Fatalf(unableToCreateTempFileErr, err)
	}
	createChildDir("cmd")
	createChildDir(firstEntryPoint)
	createChildDir(secondEntryPoint)

	const contents = `package main

import "os"

func main() {
	os.Exit(0)
}
`
	if err := os.WriteFile(filepath.Join(tempDir, firstEntryPoint, "main.go"), []byte(contents), 0644); err != nil {
		t.Fatalf(unableToCreateTempFileErr, err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, secondEntryPoint, "entry-point.go"), []byte(contents), 0644); err != nil {
		t.Fatalf(unableToCreateTempFileErr, err)
	}

	return tempDir
}

func TestGolangBOMCollection(t *testing.T) {
	testRepo := setupGolangTestRepo(t)
	defer os.RemoveAll(testRepo)

	setupMockExecutor := func() *mockCLIExecutor {
		expectedOutput, err := ioutil.ReadFile("testdata/golang-expected-bom.json")
		if err != nil {
			t.Fatalf("unable to read a test file %s", err)
		}

		executor := new(mockCLIExecutor)
		expectedCmd := fmt.Sprintf(cyclonedxGoModTemplate, firstEntryPoint)
		executor.On("shellOut", testRepo, expectedCmd).Return(string(expectedOutput), nil)
		expectedCmd = fmt.Sprintf(cyclonedxGoModTemplate, secondEntryPoint)
		executor.On("shellOut", testRepo, expectedCmd).Return(string(expectedOutput), nil)
		return executor
	}

	executor := setupMockExecutor()
	collector := Golang{executor: executor}

	got, err := Collect(collector, testRepo)

	assert.NoError(t, err)
	executor.AssertExpectations(t)
	assertGolangExpectedBOM(t, got)
}

func TestFallbackGolangBOMCollection(t *testing.T) {
	testRepo := setupGolangTestRepo(t)
	defer os.RemoveAll(testRepo)

	t.Run("use cdxgen to generate BOM if cyclonedx-gomod fails", func(t *testing.T) {
		expectedOutput, err := ioutil.ReadFile("testdata/golang-expected-bom.json")
		if err != nil {
			t.Fatalf("unable to read a test file %s", err)
		}
		executor := new(mockCLIExecutor)
		cyclonedxGoModCmd := fmt.Sprintf(cyclonedxGoModTemplate, firstEntryPoint)
		executor.On("shellOut", testRepo, cyclonedxGoModCmd).Return("", exec.ErrNotFound)
		executor.On("executeCDXGen", testRepo, cdxgenCmd).Return(string(expectedOutput), nil)

		got, err := Collect(Golang{executor: executor}, testRepo)
		assert.NoError(t, err)
		executor.AssertExpectations(t)
		assertGolangExpectedBOM(t, got)
	})

	t.Run("return an error when BOM generation fails for cyclonedx-gomod and cdxgen", func(t *testing.T) {
		executor := new(mockCLIExecutor)
		cyclonedxGoModCmd := fmt.Sprintf(cyclonedxGoModTemplate, firstEntryPoint)
		executor.On("shellOut", testRepo, cyclonedxGoModCmd).Return("", exec.ErrNotFound)
		executor.On("executeCDXGen", testRepo, cdxgenCmd).Return("", exec.ErrNotFound)

		got, err := Collect(Golang{executor: executor}, testRepo)

		assert.Empty(t, got)
		assert.ErrorIs(t, errUnsupportedRepo, err)
		executor.AssertExpectations(t)
	})

	t.Run("use cdxgen when no entry golang entry points are found", func(t *testing.T) {
		tempDir := createTempDir(t)

		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), nil, 0644); err != nil {
			t.Fatalf(unableToCreateTempFileErr, err)
		}
		expectedOutput, err := ioutil.ReadFile("testdata/golang-expected-bom.json")
		if err != nil {
			t.Fatalf("unable to read a test file %s", err)
		}

		executor := new(mockCLIExecutor)
		executor.On("executeCDXGen", tempDir, cdxgenCmd).Return(string(expectedOutput), nil)

		got, err := Collect(Golang{executor: executor}, tempDir)
		assert.NoError(t, err)
		executor.AssertExpectations(t)
		assertGolangExpectedBOM(t, got)
	})
}

func assertGolangExpectedBOM(t *testing.T, got *cdx.BOM) {
	actualBOM, err := CdxToBOMString(JSON, got)
	assert.NoError(t, err)
	assertGeneratedBOM(t, actualBOM, "testdata/golang-expected-bom.json")
}
