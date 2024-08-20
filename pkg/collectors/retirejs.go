package collectors

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/sbomsftw/pkg/bomtools"
)

type RetireJS struct{}

// GenerateBOM implements Collector interface.
func (r RetireJS) GenerateBOM(ctx context.Context, repositoryPath string) (*cdx.BOM, error) {
	// Retire JS outputs results to stderr by default - redirect to stdout with 2>&1
	const cmdTemplate = "retire --jspath %s --outputformat cyclonedx --exitwith 0 2>&1"
	re := regexp.MustCompile(`^[\w./-]*$`)

	if !re.MatchString(repositoryPath) {
		return nil, errors.New("invalid shell command")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)

	defer cancel()

	out, err := exec.CommandContext(ctx, "bash", "-c", fmt.Sprintf(cmdTemplate, repositoryPath)).Output()
	if err != nil {
		return nil, err
	}

	return bomtools.XMLStringToJSONCDX(out)
}

// String implements Collector interface.
func (r RetireJS) String() string {
	return "generic retireJS collector"
}
