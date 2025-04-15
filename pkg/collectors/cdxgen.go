package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/sbomsftw/pkg/bomtools"
)

type CDXGen struct{}

func (c CDXGen) GenerateBOM(ctx context.Context, repositoryPath string) (*cdx.BOM, error) {
	f, err := os.CreateTemp("/tmp", "cdxgen-collector-tmp-output-")
	if err != nil {
		return nil, fmt.Errorf("can't create a temp file for writing cdxgen output %v", err)
	}

	defer func() {
		_ = os.Remove(f.Name())
		_ = os.Remove(f.Name() + ".xml")
		_ = os.Remove(f.Name() + ".json")
	}()

	outputFile := f.Name() + ".json"

	cdxgenCmd := fmt.Sprintf("export FETCH_LICENSE=false && cdxgen --recursive -o %s", outputFile)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
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
