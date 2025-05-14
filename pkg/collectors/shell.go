package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/codeskyblue/go-sh"

	"github.com/CycloneDX/cyclonedx-go"
	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
	"github.com/vinted/sbomsftw/pkg/bomtools"
)

type shellExecutor interface {
	shellOut(ctx context.Context, bomRoot string, bootstrapCmd string) error
	bomFromCdxgen(ctx context.Context, bomRoot, language string, multiModuleMode bool) (*cdx.BOM, error)
}

type defaultShellExecutor struct{}

func (d defaultShellExecutor) bomFromCdxgen(
	ctx context.Context,
	bomRoot string,
	language string,
	multiModuleMode bool,
) (*cdx.BOM, error) {
	f, err := os.CreateTemp("/tmp", "sa-collector-tmp-output-")
	if err != nil {
		return nil, fmt.Errorf("can't create a temp file for writing cdxgen output: %v", err)
	}

	defer func() {
		_ = os.Remove(f.Name())
		_ = os.Remove(f.Name() + ".xml")
		_ = os.Remove(f.Name() + ".json")
	}()

	outputFile := f.Name() + ".json"

	withLicencesCommand := formatCommand(multiModuleMode, true, language, outputFile)
	sbom, err := generate(ctx, bomRoot, outputFile, withLicencesCommand, 15*time.Minute)

	if err == nil {
		return sbom, nil
	}

	log.WithError(err).
		Debug("Failed to generate SBOMs with licensing information. Attempting to generate SBOMs without licensing information.")

	withoutLicencesCommand := formatCommand(multiModuleMode, false, language, outputFile)
	sbom, err = generate(ctx, bomRoot, outputFile, withoutLicencesCommand, 10*time.Minute)
	if err == nil {
		return sbom, nil
	}

	log.WithError(err).
		Debug("Failed to generate SBOMs with and without licensing information.")

	return nil, err
}

func generate(
	ctx context.Context,
	directory string,
	outputFile string,
	command string,
	timeout time.Duration,
) (*cyclonedx.BOM, error) {

	// FIXME: This does absolutely nothing. go-sh does
	// not accept context, which means the commands
	// ran via it cannot be cancelled.
	_, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := run(directory, command)
	if err != nil {
		return nil, err
	}

	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		return nil, fmt.Errorf("SBOMs failed to generate: %s", command)
	}

	return bomtools.StringToCDX(output)
}

func run(directory string, command string) error {
	return sh.NewSession().
		SetDir(directory).
		Command("bash", "-c", command).
		Run()
}

func formatCommand(
	multiModuleMode bool,
	fetchLicense bool,
	language string,
	outputFile string,
) string {
	licenseConfig := fmt.Sprintf("export FETCH_LICENSE=%t", fetchLicense)

	multiModuleModeConfig := "unset GRADLE_MULTI_PROJECT_MODE"
	if multiModuleMode {
		multiModuleModeConfig = "export GRADLE_MULTI_PROJECT_MODE=1"
	}

	formattedCmd := fmt.Sprintf(
		"%s && %s && cdxgen --type %s -o %s",
		licenseConfig,
		multiModuleModeConfig,
		language,
		outputFile,
	)

	log.Warnf("running following cmd %s", formattedCmd)
	return formattedCmd
}

func (d defaultShellExecutor) shellOut(ctx context.Context, execDir, shellCmd string) error {
	const shellCmdTimeout = 10
	ctx, cancel := context.WithTimeout(ctx, time.Duration(shellCmdTimeout)*time.Minute)

	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd) // User controller input doesn't go here

	cmd.Dir = execDir

	return cmd.Run()
}
