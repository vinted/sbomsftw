package collectors

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/mock"
)

type mockShellExecutor struct{ mock.Mock }

func (m *mockShellExecutor) shellOut(bomRoot string, shellCmd string) error {
	return m.Called(bomRoot, shellCmd).Error(0)
}

func (m *mockShellExecutor) bomFromCdxgen(bomRoot, language string, multiModuleMode bool) (*cdx.BOM, error) {
	args := m.Called(bomRoot, language, multiModuleMode)
	return args.Get(0).(*cdx.BOM), args.Error(1)
}
