package collectors

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	bootstrapFailedErr = "can't boostrap %s: %w" //Used whenever bundler install fails
)

type CLIExecutor interface {
	bootstrap(bomRoot string, bootstrapCmd string) error
	executeCDXGen(bomRoot string, cdxgenCmd string) (string, error)
}

type defaultCLIExecutor struct{}

func (e defaultCLIExecutor) executeCDXGen(bomRoot, shellCMD string) (string, error) {
	cmd := exec.Command("bash", "-c", shellCMD)
	cmd.Dir = bomRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("can't collect BOM for %s: %w", bomRoot, err)
	}
	//cdxgen even on failure exits with 0 status code. This is kinda ugly but must compare stdout :/
	if strings.HasPrefix(string(out), "Unable to produce BOM") {
		errMsg := fmt.Sprintf("can't collect BOM for %s: %s", bomRoot, string(out))
		return "", BOMCollectionFailed(errMsg)
	}
	return string(out), nil
}

func (e defaultCLIExecutor) bootstrap(bomRoot, bootstrapCmd string) error {
	cmd := exec.Command("bash", "-c", bootstrapCmd)
	cmd.Dir = bomRoot
	return cmd.Run()
}
