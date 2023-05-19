package collectors

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonCollector(t *testing.T) {
	createTempDir := func(t *testing.T) string {
		tempDirName, err := os.MkdirTemp("/tmp", "sa")
		if err != nil {
			t.Fatalf("unable to create temp directory for testing: %s", err)
		}

		return tempDirName
	}

	t.Run("bootstrap language files correctly", func(t *testing.T) {
		bomRoots := []string{
			"/tmp/some-random-dir/requirements.txt",
			"/tmp/some-random-dir/setup.py",
			"/tmp/some-random-dir/inner-dir/Pipfile.lock",
			"/tmp/some-random-dir/inner-dir/deepest-dir/poetry.lock",
		}
		got := Python{}.BootstrapLanguageFiles(context.Background(), bomRoots)
		assert.ElementsMatch(t, bomRoots, got)
	})

	t.Run("generate BOM correctly", func(t *testing.T) {
		const bomRoot = "/tmp/some-random-dir"
		executor := new(mockShellExecutor)
		executor.On("bomFromCdxgen", bomRoot, "python", false).Return(new(cdx.BOM), nil)
		_, _ = Python{executor: executor}.GenerateBOM(context.Background(), "/tmp/some-random-dir/setup.py")
		executor.AssertExpectations(t)
	})

	t.Run("match correct package files", func(t *testing.T) {
		pythonCollector := Python{}

		packageFiles := []string{
			"setup.py",
			"requirements.txt",
			"/opt/Pipfile.lock",
			"/opt/poetry.lock",
		}

		for _, f := range packageFiles {
			assert.True(t, pythonCollector.MatchLanguageFiles(false, f))
		}

		condaEnvFiles := []string{
			"environment.yml",
			"environment.yaml",
			"environment-server.yml",
			"/opt/environment_3.7.yml",
			"/opt/environment-3.8.yml",
		}
		for _, f := range condaEnvFiles {
			assert.True(t, pythonCollector.MatchLanguageFiles(false, f))
		}

		for _, f := range []string{"setup.py", "requirements.txt", "Pipfile.lock", "poetry.lock"} {
			assert.False(t, pythonCollector.MatchLanguageFiles(true, f))
		}
		assert.False(t, pythonCollector.MatchLanguageFiles(false, "environment-dev.yaml"))
		assert.False(t, pythonCollector.MatchLanguageFiles(false, "/etc/passwd"))
	})

	t.Run("don't create requirements.txt if no conda environment files exist", func(t *testing.T) {
		tempDir := createTempDir(t)
		err := os.WriteFile(filepath.Join(tempDir, "setup.py"), nil, 0o644)
		require.NoError(t, err)

		Python{}.BootstrapLanguageFiles(context.Background(), []string{filepath.Join(tempDir, "setup.py")})
		_, err = os.ReadFile(filepath.Join(tempDir, "requirements.txt"))
		// assert.NotNil(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("convert conda environments to requirements.txt correctly", func(t *testing.T) {
		/*
			Setup test environment. Setup function creates the following temp directory structure:
			/tmp/random-dir/environment_3.7.yml
			/tmp/random-dir/environment_3.8.yml
			/tmp/random-dir/inner-dir/environment.yml
			/tmp/random-dir/inner-dir/requirements.txt
		*/
		setup := func() (string, string) {
			tempDir := createTempDir(t)

			contents, err := os.ReadFile("../../integration/test/conda-envs/test-conda-env3.7.yml")
			require.NoError(t, err)

			err = os.WriteFile(filepath.Join(tempDir, "environment_3.7.yml"), contents, 0o644)
			require.NoError(t, err)

			contents, err = os.ReadFile("../../integration/test/conda-envs/test-conda-env3.8.yml")
			require.NoError(t, err)

			err = os.WriteFile(filepath.Join(tempDir, "environment_3.8.yml"), contents, 0o644)
			require.NoError(t, err)

			innerDir, err := os.MkdirTemp(tempDir, "innerDir")
			require.NoError(t, err)

			contents, err = os.ReadFile("../../integration/test/conda-envs/test-conda-env.yml")
			require.NoError(t, err)

			err = os.WriteFile(filepath.Join(innerDir, "environment.yml"), contents, 0o644)
			require.NoError(t, err)

			contents, err = os.ReadFile("../../integration/test/conda-envs/test-reqs.txt")
			require.NoError(t, err)

			err = os.WriteFile(filepath.Join(innerDir, "requirements.txt"), contents, 0o644)
			require.NoError(t, err)

			return tempDir, innerDir
		}

		tempDir, innerDir := setup()
		defer os.RemoveAll(tempDir)

		bomRoots := []string{
			filepath.Join(tempDir, "environment_3.7.yml"),
			filepath.Join(tempDir, "environment_3.8.yml"),
			filepath.Join(innerDir, "environment.yml"),
		}

		Python{}.BootstrapLanguageFiles(context.Background(), bomRoots)

		got, err := os.ReadFile(filepath.Join(tempDir, "requirements.txt"))
		require.NoError(t, err)
		want := `Fastapi==0.65.1
gunicorn==20.1.0
json-logging==1.4.0rc1
kafka-python==2.0.2
pip==20.0.2
prometheus-client==0.9.0
python-dotenv==0.17.1
python==3.7.6
python==3.8.12
statsd==3.3.0
uvicorn==0.13.4`
		assert.Equal(t, want, string(got))

		got, err = os.ReadFile(filepath.Join(innerDir, "requirements.txt"))
		require.NoError(t, err)
		want = `gcsfs==0.6.0
google-cloud-storage==1.24.1
gunicorn==20.1.0
json-logging==1.4.0rc1
pip==20.0.2
prometheus-client==0.9.0
pydantic==1.6.1
python==3.7.6`
		assert.Equal(t, want, string(got))
	})
}
