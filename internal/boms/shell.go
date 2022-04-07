package boms

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

type UnsafeShellOutError struct{ cmd string }

func (e UnsafeShellOutError) Error() string {
	return fmt.Sprintf("attempting to shell out with unsafe input: %s", e.cmd)
}

type CLIExecutor interface {
	shellOut(bomRoot string, bootstrapCmd string) (string, error)
	bomFromCdxgen(bomRoot string, language language) (*cdx.BOM, error)
}

type defaultCLIExecutor struct{}

func bomFromTrivy(repositoryPath string) (*cdx.BOM, error) {
	cmd := fmt.Sprintf("trivy --quiet fs --format cyclonedx %s", repositoryPath)
	re := regexp.MustCompile(`^[\w./-]*$`)

	if matches := re.MatchString(repositoryPath); !matches { //Sanitize repositoryPath given to prevent cmd injection
		return nil, UnsafeShellOutError{cmd: cmd}
	}
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return nil, err
	}
	return BomStringToCDX(JSON, string(out))
}

func (e defaultCLIExecutor) bomFromCdxgen(bomRoot string, language language) (*cdx.BOM, error) {
	const cdxgenTemplate = "cdxgen --type %s -o %s"

	f, err := ioutil.TempFile("/tmp", "sa-collector-tmp-output-")
	if err != nil {
		return nil, fmt.Errorf("can't create a temp file for writing cdxgen output %w", err)
	}
	// CDXGen creates multiple files on success, one with .xml extension and another with .json extension
	defer os.Remove(f.Name())
	defer os.Remove(f.Name() + ".xml")
	defer os.Remove(f.Name() + ".json")
	outputFile := f.Name() + ".json"

	//Execute cdxgen
	cmd := exec.Command("bash", "-c", fmt.Sprintf(cdxgenTemplate, language, outputFile))
	cmd.Dir = bomRoot
	if cmd.Run() != nil {
		return nil, fmt.Errorf("can't Collect BOM for %s: %w", bomRoot, err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		return nil, fmt.Errorf("can't Collect %s BOM for %s", language, bomRoot)
	}
	return BomStringToCDX(JSON, string(output))
}

//todo add sanitization
func (e defaultCLIExecutor) shellOut(execDir, shellCmd string) (string, error) {
	cmd := exec.Command("bash", "-c", shellCmd)
	cmd.Dir = execDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), err
}
