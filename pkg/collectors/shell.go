package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
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

	f, err := os.CreateTemp("/tmp", "sa-collector-tmp-output-")
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
	var (
		withLicensesTimeout    = time.Duration(15) * time.Minute
		withoutLicensesTimeout = time.Duration(10) * time.Minute
	)
	// Fetching licenses can time out so add a cancellation of 15 minutes
	withLicensesCtx, withLicensesCancel := context.WithTimeout(ctx, withLicensesTimeout)
	defer withLicensesCancel()

	cdxGenCmd := formatCDXGenCmd(multiModuleMode, true, language, outputFile)
	cmd := exec.CommandContext(withLicensesCtx, "bash", "-c", cdxGenCmd)
	cmd.Dir = bomRoot

	// Set process group for better process management
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	pgid := int(0) // Will store process group ID for cleanup

	// Run the command and get the process group ID
	if err = cmd.Start(); err == nil {
		pgid = cmd.Process.Pid // Save PID for process group termination
		err = cmd.Wait()
	}

	// Ensure any spawned processes are terminated
	if pgid > 0 {
		// Kill the entire process group
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}

	// If first attempt failed, try without licenses
	if err != nil {
		log.WithError(err).Debugf("cdxgen failed - regenerating SBOMs without licensing info")

		withoutLicensesCtx, withoutLicensesCancel := context.WithTimeout(ctx, withoutLicensesTimeout)
		defer withoutLicensesCancel()

		cdxGenCmd = formatCDXGenCmd(multiModuleMode, false, language, outputFile)
		cmd = exec.CommandContext(withoutLicensesCtx, "bash", "-c", cdxGenCmd)
		cmd.Dir = bomRoot
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		pgid = 0 // Reset process group ID

		if err = cmd.Start(); err == nil {
			pgid = cmd.Process.Pid
			err = cmd.Wait()
		}

		// Ensure any spawned processes are terminated
		if pgid > 0 {
			// Kill the entire process group
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}

		if err != nil {
			return nil, fmt.Errorf("can't Collect SBOMs for %s: %v", bomRoot, err)
		}
	}

	// Read and process output
	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		return nil, fmt.Errorf("can't Collect %s SBOMs for %s", language, bomRoot)
	}

	return bomtools.StringToCDX(output)
}

func (d defaultShellExecutor) shellOut(ctx context.Context, execDir, shellCmd string) error {
	const shellCmdTimeout = 10
	ctx, cancel := context.WithTimeout(ctx, time.Duration(shellCmdTimeout)*time.Minute)

	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd) // User controller input doesn't go here

	cmd.Dir = execDir

	return cmd.Run()
}
