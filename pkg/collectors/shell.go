package collectors

import (
	"context"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"os"
	"os/exec"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
	"github.com/vinted/sbomsftw/pkg/bomtools"
)

type shellExecutor interface {
	shellOut(ctx context.Context, bomRoot string, bootstrapCmd string) error
	bomFromCdxgen(ctx context.Context, bomRoot, language string, multiModuleMode bool) (*cdx.BOM, error)
}

type defaultShellExecutor struct{}

func (d defaultShellExecutor) bomFromCdxgen(ctx context.Context, bomRoot string, language string, multiModuleMode bool) (*cdx.BOM, error) {
	formatCDXGenCmd := func(multiModuleMode, fetchLicense bool, language, outputFile string) string {
		licenseConfig := fmt.Sprintf("export FETCH_LICENSE=%t", fetchLicense)

		var multiModuleModeConfig string
		if multiModuleMode {
			multiModuleModeConfig = "export GRADLE_MULTI_PROJECT_MODE=1"
		} else {
			multiModuleModeConfig = "unset GRADLE_MULTI_PROJECT_MODE"
		}
		formattedCmd := fmt.Sprintf("%s && %s && cdxgen --type %s -o %s", licenseConfig, multiModuleModeConfig, language, outputFile)
		log.Warnf("running following cmd %s", formattedCmd)
		return formattedCmd
	}

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

	withLicensesTimeout := 15 * time.Minute
	withoutLicensesTimeout := 10 * time.Minute

	_, withLicensesCancel := context.WithTimeout(ctx, withLicensesTimeout)
	defer withLicensesCancel()

	if err := runCDXGenCommand(bomRoot, formatCDXGenCmd(multiModuleMode, true, language, outputFile)); err != nil {
		log.WithError(err).Debugf("cdxgen failed - regenerating SBOMs without licensing info")

		_, withoutLicensesCancel := context.WithTimeout(ctx, withoutLicensesTimeout)
		defer withoutLicensesCancel()

		if err := runCDXGenCommand(bomRoot, formatCDXGenCmd(multiModuleMode, false, language, outputFile)); err != nil {
			return nil, fmt.Errorf("can't collect SBOMs for %s: %v", bomRoot, err)
		}
	}

	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		return nil, fmt.Errorf("can't collect %s SBOMs for %s", language, bomRoot)
	}

	return bomtools.StringToCDX(output)
}

func runCDXGenCommand(dir, cmd string) error {
	return sh.NewSession().SetDir(dir).Command("bash", "-c", cmd).Run()
}

func (d defaultShellExecutor) shellOut(ctx context.Context, execDir, shellCmd string) error {
	const shellCmdTimeout = 10
	ctx, cancel := context.WithTimeout(ctx, time.Duration(shellCmdTimeout)*time.Minute)

	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd) // User controller input doesn't go here

	cmd.Dir = execDir

	return cmd.Run()
}
