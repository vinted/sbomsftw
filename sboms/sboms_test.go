package sboms_test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vinted/software-assets/sboms"
	"io/fs"
	"testing"
	"testing/fstest"
)

const (
	unableToGenerateBOMMsg        = "no Lockfile present"
	unableToOpenPathMsg           = "unable to open the provided path"
	expectedBOMGenerationErrorMsg = "BOM Generation for Mock Handler failed! Reason: " + unableToGenerateBOMMsg
	expectedTraversalErrorMsg     = "FS Traversal using Mock Handler failed! Reason: " + unableToOpenPathMsg
)

type MockLogger struct{ mock.Mock }

type MockHandler struct {
	mock.Mock
	FileMatchAttempts []string
}

func (m *MockLogger) LogError(err error) {
	m.Called(err)
}

func (m *MockLogger) LogMessage(bomRoot string) {
	m.Called(bomRoot)
}

func (m *MockHandler) MatchFile(filename string) bool {
	m.FileMatchAttempts = append(m.FileMatchAttempts, filename)
	return filename == "Packages" || filename == "Packages.lock"
}

func (m *MockHandler) GenerateBOM(bomRoot string) (string, error) {
	args := m.Called(bomRoot)
	return args.String(0), args.Error(1)
}

func (m *MockHandler) String() string {
	return "Mock Handler"
}

var testFS = fstest.MapFS{
	"test-repository/Packages":                            {},
	"test-repository/ignore.txt":                          {},
	"test-repository/Packages.lock":                       {},
	"test-repository/inner-dir":                           {Mode: fs.ModeDir},
	"test-repository/inner-dir/Packages":                  {Mode: fs.ModeDir},
	"test-repository/inner-dir/Packages.lock":             {Mode: fs.ModeDir},
	"test-repository/inner-dir/deepest-dir/Packages.lock": {},
}
var expectedBOMRoots = []string{
	"test-repository/Packages",
	"test-repository/Packages.lock",
	"test-repository/inner-dir/deepest-dir/Packages.lock",
}

func TestBOMCollection(t *testing.T) {
	t.Run("finds correct BOM roots based on the handler provided", func(t *testing.T) {

		logger := new(MockLogger)
		handler := new(MockHandler)

		for _, bomRoot := range expectedBOMRoots {
			handler.On("GenerateBOM", bomRoot).Return("", nil)
		}

		_ = sboms.Collect(testFS, handler, logger)
		handler.AssertExpectations(t)

		assert.Equal(t, []string{".", "test-repository", "Packages", "Packages.lock", "ignore.txt", "inner-dir",
			"Packages", "Packages.lock", "deepest-dir", "Packages.lock"}, handler.FileMatchAttempts)
	})
}

type MockFS struct{ mock.Mock }

func (m *MockFS) Open(name string) (fs.File, error) {
	args := m.Called(name)
	file := args.Get(0)
	if file == nil {
		return nil, args.Error(1)
	}
	return file.(fs.File), args.Error(1)
}

func TestErrorLogging(t *testing.T) {
	t.Run("errors are logged whenever BOM generation fails", func(t *testing.T) {
		logger := new(MockLogger)
		handler := new(MockHandler)

		for _, bomRoot := range expectedBOMRoots {
			handler.On("GenerateBOM", bomRoot).Return("", errors.New(unableToGenerateBOMMsg))
		}
		logger.On("LogError", errors.New(expectedBOMGenerationErrorMsg)).Return()

		_ = sboms.Collect(testFS, handler, logger)

		handler.AssertExpectations(t)

		logger.AssertExpectations(t)
		logger.AssertNumberOfCalls(t, "LogError", 3)
	})

	t.Run("error is logged whenever file-system walk fails", func(t *testing.T) {
		mockFS := new(MockFS)
		logger := new(MockLogger)
		handler := new(MockHandler)

		mockFS.On("Open", ".").Return(nil, errors.New(unableToOpenPathMsg))
		logger.On("LogError", errors.New(expectedTraversalErrorMsg)).Return()
		got := sboms.Collect(mockFS, handler, logger)

		assert.Empty(t, got)

		mockFS.AssertExpectations(t)
		mockFS.AssertNumberOfCalls(t, "Open", 1)

		logger.AssertExpectations(t)
		logger.AssertNumberOfCalls(t, "LogError", 1)

		assert.Empty(t, handler.FileMatchAttempts)
	})
}
