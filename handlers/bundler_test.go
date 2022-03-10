package handlers_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vinted/software-assets/handlers"
	"os"
	"testing"
)

const gemfile = "Gemfile"
const gemfileLock = "Gemfile.lock"

type MockCLIExecutor struct{ mock.Mock }

func (m *MockCLIExecutor) Cd(dir string) error {
	args := m.Called(dir)
	return args.Error(0)
}

func (m *MockCLIExecutor) ShellOut(cmd string) (string, error) {
	args := m.Called(cmd)
	return args.String(0), args.Error(1)
}

func TestMatchFile(t *testing.T) {
	bundler := handlers.Bundler{}
	assert.True(t, bundler.MatchFile(gemfile))
	assert.True(t, bundler.MatchFile(gemfileLock))
	assert.False(t, bundler.MatchFile("/etc/passwd"))
}

func TestGenerateBOM(t *testing.T) {

	bootstrapCmd := "bundler install"
	bomRoot := "/tmp/checkouts/sample-code-ruby"
	bomGenCmd := "cyclonedx-ruby -p " + bomRoot + " --output /dev/stdout 2>/dev/null | sed '$d'"

	t.Run("create Gemfile.lock before BOM generation if no lockfile is present", func(t *testing.T) {
		shelly := new(MockCLIExecutor)
		shelly.On("Cd", bomRoot).Return(nil)
		shelly.On("ShellOut", bootstrapCmd).Return("✅", nil)
		shelly.On("ShellOut", bomGenCmd).Return("👌", nil)

		got, err := handlers.Bundler{Executor: shelly}.GenerateBOM(bomRoot + "/" + gemfile)

		assert.NoError(t, err)
		assert.Equal(t, "👌", got)
		shelly.AssertExpectations(t)
	})

	t.Run("don't run bundler install if directory change is unsuccessful", func(t *testing.T) {
		shelly := new(MockCLIExecutor)
		shelly.On("Cd", bomRoot).Return(os.ErrNotExist)

		got, err := handlers.Bundler{Executor: shelly}.GenerateBOM(bomRoot + "/" + gemfile)

		assert.Empty(t, got)
		assert.Error(t, err, os.ErrNotExist)
	})

	t.Run("don't generate bom if bundler install failed", func(t *testing.T) {
		shelly := new(MockCLIExecutor)
		shelly.On("Cd", bomRoot).Return(nil)
		shelly.On("ShellOut", bootstrapCmd).Return("", os.ErrPermission)

		got, err := handlers.Bundler{Executor: shelly}.GenerateBOM(bomRoot + "/" + gemfile)

		assert.Empty(t, got)
		assert.Error(t, err, os.ErrPermission)
		shelly.AssertExpectations(t)
	})

	t.Run("don't create Gemfile.lock if it already exists", func(t *testing.T) {
		shelly := new(MockCLIExecutor)
		shelly.On("ShellOut", bomGenCmd).Return("👌", nil)

		got, err := handlers.Bundler{Executor: shelly}.GenerateBOM(bomRoot + "/" + gemfileLock)

		assert.NoError(t, err)
		assert.Equal(t, "👌", got)
		shelly.AssertExpectations(t)
	})
}
