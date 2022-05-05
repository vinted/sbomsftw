package collectors

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
	"github.com/vinted/software-assets/pkg/bomtools"
)

type shellExecutor interface {
	shellOut(bomRoot string, bootstrapCmd string) error
	bomFromCdxgen(bomRoot, language string, multiModuleMode bool) (*cdx.BOM, error)
}

type defaultShellExecutor struct {
	ctx context.Context
}

func newDefaultShellExecutor(ctx context.Context) defaultShellExecutor {
	return defaultShellExecutor{ctx: ctx}
}

func (d defaultShellExecutor) bomFromCdxgen(bomRoot string, language string, multiModuleMode bool) (*cdx.BOM, error) {
	formatCDXGenCmd := func(multiModuleMode, fetchLicense bool, language, outputFile string) string {
		licenseConfig := fmt.Sprintf("export FETCH_LICENSE=%t", fetchLicense)

		var (
			multiModuleModeConfig string
			template              = "%s && %s && cdxgen --type %s -o %s"
		)

		if multiModuleMode {
			multiModuleModeConfig = "export GRADLE_MULTI_PROJECT_MODE=1"
		} else {
			multiModuleModeConfig = "unset GRADLE_MULTI_PROJECT_MODE"
		}

		return fmt.Sprintf(template, licenseConfig, multiModuleModeConfig, language, outputFile)
	}

	f, err := ioutil.TempFile("/tmp", "sa-collector-tmp-output-")
	if err != nil {
		return nil, fmt.Errorf("can't create a temp file for writing cdxgen output %v", err)
	}
	// Cleanup func. CDXGen creates multiple files on success, even if we only ask for one
	defer func() {
		// Ignore errors because when cdxgen fails it creates no files for us to remove
		_ = os.Remove(f.Name())
		_ = os.Remove(f.Name() + ".xml")
		_ = os.Remove(f.Name() + ".json")
	}()

	outputFile := f.Name() + ".json"

	// Timeouts for SBOM generation with CDXGen
	const (
		generateSBOMsWithLicenses    = 15
		generateSBOMsWithoutLicenses = 10
	)
	// Fetching licenses can time out so add a cancellation of 15 minutes
	ctx, cancel := context.WithTimeout(d.ctx, time.Duration(generateSBOMsWithLicenses)*time.Minute)
	cdxGenCmd := formatCDXGenCmd(multiModuleMode, true, language, outputFile)
	cmd := exec.CommandContext(ctx, "bash", "-c", cdxGenCmd) //nolint:gosec
	cmd.Dir = bomRoot

	if err = cmd.Run(); err != nil {
		cancel()
		log.WithError(err).Debugf("cdxgen failed - regenerating SBOMs without licensing info")
		ctx, cancel = context.WithTimeout(d.ctx, time.Duration(generateSBOMsWithoutLicenses)*time.Minute)
		cdxGenCmd = formatCDXGenCmd(multiModuleMode, false, language, outputFile)
		cmd = exec.CommandContext(ctx, "bash", "-c", cdxGenCmd) //nolint:gosec
		cmd.Dir = bomRoot

		if err = cmd.Run(); err != nil {
			cancel()

			return nil, fmt.Errorf("can't Collect BOM for %s: %v", bomRoot, err)
		}
		cancel()
	}
	cancel()

	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		return nil, fmt.Errorf("can't Collect %s BOM for %s", language, bomRoot)
	}

	return bomtools.StringToCDX(output)
}

func (d defaultShellExecutor) shellOut(execDir, shellCmd string) error {
	const shellCmdTimeout = 10
	ctx, cancel := context.WithTimeout(d.ctx, time.Duration(shellCmdTimeout)*time.Minute)

	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd) // User controller input doesn't go here

	cmd.Dir = execDir

	return cmd.Run()
}
