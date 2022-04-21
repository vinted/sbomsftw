package collectors

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
)

type ShellExecutor interface {
	bomFromCdxgen(bomRoot, language string, multiModuleMode bool) (*cdx.BOM, error)
	shellOut(bomRoot string, bootstrapCmd string) error
}

type DefaultShellExecutor struct{}

func (d DefaultShellExecutor) bomFromCdxgen(bomRoot string, language string, multiModuleMode bool) (*cdx.BOM, error) {

	formatCDXGenCmd := func(multiModuleMode, fetchLicense bool, language, outputFile string) string {
		licenseConfig := fmt.Sprintf("export FETCH_LICENSE=%t", fetchLicense)
		var multiModuleModeConfig string
		if multiModuleMode {
			multiModuleModeConfig = "export GRADLE_MULTI_PROJECT_MODE=1"
		} else {
			multiModuleModeConfig = "unset GRADLE_MULTI_PROJECT_MODE"
		}
		const template = "%s && %s && cdxgen --type %s -o %s"
		return fmt.Sprintf(template, licenseConfig, multiModuleModeConfig, language, outputFile)
	}

	f, err := ioutil.TempFile("/tmp", "sa-collector-tmp-output-")
	if err != nil {
		return nil, fmt.Errorf("can't create a temp file for writing cdxgen output %v", err)
	}
	//Cleanup func. CDXGen creates multiple files on success, even if we only ask for one
	defer func() {
		//Ignore errors because when cdxgen fails it creates no files for us to remove
		_ = os.Remove(f.Name())
		_ = os.Remove(f.Name() + ".xml")
		_ = os.Remove(f.Name() + ".json")
	}()

	outputFile := f.Name() + ".json"

	//Fetching licenses can timeout so add a cancelation of 15 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	cmd := exec.CommandContext(ctx, "bash", "-c", formatCDXGenCmd(multiModuleMode, true, language, outputFile))
	cmd.Dir = bomRoot

	if err = cmd.Run(); err != nil {
		cancel()
		log.WithField("error", err).Errorf("cdxgen failed - regenerating SBOMs without licensing info")
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		cmd = exec.CommandContext(ctx, "bash", "-c", formatCDXGenCmd(multiModuleMode, false, language, outputFile))
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

func (d DefaultShellExecutor) shellOut(execDir, shellCmd string) error {
	cmd := exec.Command("bash", "-c", shellCmd)
	cmd.Dir = execDir
	return cmd.Run()
}
