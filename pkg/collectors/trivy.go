package collectors

import (
	"errors"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
	"os/exec"
	"regexp"
)

type Trivy struct{}

//GenerateBOM TODO Refactor this and don't shell out
func (t Trivy) GenerateBOM(repositoryPath string) (*cdx.BOM, error) {
	cmd := fmt.Sprintf("trivy --quiet fs --format cyclonedx %s", repositoryPath)
	re := regexp.MustCompile(`^[\w./-]*$`)

	if matches := re.MatchString(repositoryPath); !matches { //Sanitize repositoryPath given to prevent cmd injection
		return nil, errors.New("invalid shell command")
	}
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return nil, err
	}
	return bomtools.StringToCDX(out)
}

//String implements BOMCollector interface
func (t Trivy) String() string {
	return "trivy collector"
}
