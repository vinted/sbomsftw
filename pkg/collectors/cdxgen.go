package collectors

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
)

type CDXGen struct {
	ctx context.Context
}

func NewCDXGenCollector(ctx context.Context) CDXGen {
	return CDXGen{
		ctx: ctx,
	}
}

func (c CDXGen) GenerateBOM(repositoryPath string) (*cdx.BOM, error) {
	f, err := ioutil.TempFile("/tmp", "cdxgen-collector-tmp-output-")
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

	cdxgenCmd := fmt.Sprintf("export FETCH_LICENSE=false && cdxgen --recursive -o %s", outputFile)
	ctx, cancel := context.WithTimeout(c.ctx, 15*time.Minute)

	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cdxgenCmd)
	cmd.Dir = repositoryPath

	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("can't collect BOMs for %s: %v", repositoryPath, err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil || len(output) == 0 {
		return nil, fmt.Errorf("can't collect BOMs for %s", repositoryPath)
	}

	return bomtools.StringToCDX(output)
}

// String implements BOMCollector interface
func (c CDXGen) String() string {
	return "generic cdxgen collector"
}
