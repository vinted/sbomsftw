package collectors

import (
	"context"
	"errors"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
	"os/exec"
	"regexp"
	"time"
)

type Trivy struct{}

//GenerateBOM implements Collector interface
func (t Trivy) GenerateBOM(repositoryPath string) (*cdx.BOM, error) {
	cmd := fmt.Sprintf("trivy --quiet fs --format cyclonedx %s", repositoryPath)
	re := regexp.MustCompile(`^[\w./-]*$`)

	if !re.MatchString(repositoryPath) {
		return nil, errors.New("invalid shell command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "bash", "-c", cmd).Output()
	if err != nil {
		return nil, err
	}
	return bomtools.StringToCDX(out)
}

//String implements Collector interface
func (t Trivy) String() string {
	return "generic trivy collector"
}
