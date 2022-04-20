package collectors

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
)

type ShellExecutor interface {
	bomFromCdxgen(bomRoot, language string, multiModuleMode bool) (*cdx.BOM, error)
	shellOut(bomRoot string, bootstrapCmd string) error
}

type DefaultShellExecutor struct{}

func (d DefaultShellExecutor) bomFromCdxgen(bomRoot string, language string, multiModuleMode bool) (*cdx.BOM, error) {
	var cdxgenTemplate string
	if multiModuleMode {
		cdxgenTemplate = "export GRADLE_MULTI_PROJECT_MODE=1 && cdxgen --type %s -o %s"
	} else {
		cdxgenTemplate = "unset GRADLE_MULTI_PROJECT_MODE && cdxgen --type %s -o %s"
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
	//Execute cdxgen
	cmd := exec.Command("bash", "-c", fmt.Sprintf(cdxgenTemplate, language, outputFile))
	cmd.Dir = bomRoot

	_, err = cmd.Output()

	//fmt.Println(string(out))
	if err != nil {
		return nil, fmt.Errorf("can't Collect BOM for %s: %v", bomRoot, err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		// fmt.Println("collection failed")
		// fmt.Println(err)
		return nil, fmt.Errorf("can't Collect %s BOM for %s %v", language, bomRoot, err)
	}
	return bomtools.StringToCDX(output)
}

func (d DefaultShellExecutor) shellOut(execDir, shellCmd string) error {
	cmd := exec.Command("bash", "-c", shellCmd)
	cmd.Dir = execDir
	return cmd.Run()
}
