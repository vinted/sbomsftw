package collectors

import (
	"context"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"github.com/mitchellh/go-ps"
	"os"
	"os/exec"
	"strings"
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

		return fmt.Sprintf("%s && %s && cdxgen --no-install-deps --type %s -o %s", licenseConfig, multiModuleModeConfig, language, outputFile)
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

// cleanupNewProcesses finds and terminates new Java processes that weren't running before
func cleanupNewProcesses(pidsBefore map[int]struct{}) {
	processesAfter, err := ps.Processes()
	if err != nil {
		log.WithError(err).Warn("Failed to get process list for cleanup")
		return
	}

	for _, p := range processesAfter {
		// Check if this is a new process (wasn't running before)
		if _, existed := pidsBefore[p.Pid()]; !existed {
			execName := strings.ToLower(p.Executable())
			log.Warnf("leftover pid %s", execName)

			// Check if it's a Java process or related to cdxgen
			if strings.Contains(execName, "java") ||
				strings.Contains(execName, "jvm") ||
				strings.Contains(execName, "gradle") ||
				strings.Contains(execName, "maven") ||
				strings.Contains(execName, "cdxgen") {

				log.Debugf("Terminating leftover process: %s (PID: %d)", execName, p.Pid())
				killProcess(p.Pid())
			}
		}
	}
}

// killProcess terminates a process by PID
func killProcess(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		log.WithError(err).Debugf("Could not find process %d", pid)
		return
	}

	// First try SIGTERM for graceful shutdown
	if err := proc.Signal(os.Interrupt); err != nil {
		log.WithError(err).Debugf("Failed to send SIGTERM to process %d", pid)

		// If SIGTERM fails, force kill
		if err := proc.Kill(); err != nil {
			log.WithError(err).Debugf("Failed to kill process %d", pid)
		}
	}

	// Wait for the process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	// Wait up to 5 seconds for process to exit
	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Force kill if it didn't exit
		_ = proc.Kill()
	}
}

func (d defaultShellExecutor) shellOut(ctx context.Context, execDir, shellCmd string) error {
	const shellCmdTimeout = 10
	ctx, cancel := context.WithTimeout(ctx, time.Duration(shellCmdTimeout)*time.Minute)

	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd) // User controller input doesn't go here

	cmd.Dir = execDir

	return cmd.Run()
}
