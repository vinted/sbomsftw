package boms

import (
	"fmt"
	"os/exec"
	"strings"
)

type CLIExecutor interface {
	shellOut(bomRoot string, bootstrapCmd string) (string, error)
	executeCDXGen(bomRoot string, cdxgenCmd string) (string, error)
}

type defaultCLIExecutor struct{}

func (e defaultCLIExecutor) executeCDXGen(bomRoot, shellCMD string) (string, error) {
	cmd := exec.Command("bash", "-c", shellCMD)
	cmd.Dir = bomRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("can't Collect BOM for %s: %w", bomRoot, err)
	}
	//cdxgen even on failure exits with 0 status code. This is kinda ugly but must compare stdout :/
	if strings.HasPrefix(string(out), "Unable to produce BOM") {
		errMsg := fmt.Sprintf("can't Collect BOM for %s: %s", bomRoot, string(out))
		return "", BOMCollectionFailed(errMsg)
	}
	return string(out), nil
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
