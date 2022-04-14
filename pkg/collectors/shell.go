package collectors

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
	"io/ioutil"
	"os"
	"os/exec"
)

type ShellExecutor interface {
	bomFromCdxgen(bomRoot, language string) (*cdx.BOM, error)
	shellOut(bomRoot string, bootstrapCmd string) (string, error)
}

type DefaultShellExecutor struct{}

func (d DefaultShellExecutor) bomFromCdxgen(bomRoot string, language string) (*cdx.BOM, error) {
	const cdxgenTemplate = "export FETCH_LICENSE=true && cdxgen --type %s -o %s"

	f, err := ioutil.TempFile("/tmp", "sa-collector-tmp-output-")
	if err != nil {
		return nil, fmt.Errorf("can't create a temp file for writing cdxgen output %w", err)
	}
	//Cleanup func. CDXGen creates multiple files on success, even if we only ask for one
	defer func() {
		//Ignore errors because when cdxgen fails it creates no files for us to remove
		_ = os.Remove(f.Name())
		_ = os.Remove(f.Name() + ".xml")
		_ = os.Remove(f.Name() + ".json")
	}()

	outputFile := f.Name() + ".json"

	//Execute cdxgen
	cmd := exec.Command("bash", "-c", fmt.Sprintf(cdxgenTemplate, language, outputFile))
	cmd.Dir = bomRoot
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("can't Collect BOM for %s: %w", bomRoot, err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		return nil, fmt.Errorf("can't Collect %s BOM for %s", language, bomRoot)
	}
	return bomtools.StringToCDX(output)
}

func (d DefaultShellExecutor) shellOut(execDir, shellCmd string) (string, error) {
	cmd := exec.Command("bash", "-c", shellCmd)
	cmd.Dir = execDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), err
}
