package collectors

import (
	"io/ioutil"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/mock"
)

type mockBOMBridge struct{ mock.Mock }

func (m *mockBOMBridge) shellOut(bomRoot string, shellCmd string) (string, error) {
	args := m.Called(bomRoot, shellCmd)
	return args.String(0), args.Error(1)
}

func (m *mockBOMBridge) bomFromCdxgen(bomRoot, language string, multiModuleMode bool) (*cdx.BOM, error) {
	args := m.Called(bomRoot, language, multiModuleMode)
	return args.Get(0).(*cdx.BOM), args.Error(1)
}

func createTempDir(t *testing.T) string {
	tempDirName, err := ioutil.TempDir("/tmp", "sa")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing: %s", err)
	}
	return tempDirName
}
