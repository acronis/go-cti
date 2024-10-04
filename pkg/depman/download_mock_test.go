package depman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/storage"

	"github.com/otiai10/copy"
)

type mockDownloader struct {
}

type mockInfo struct {
	Name    string
	Version string
}

func (i *mockInfo) Validate(o storage.Origin) error {
	oi, ok := o.(*mockInfo)
	if !ok {
		return fmt.Errorf("origin is not a mockInfo")
	}

	if i.Version != oi.Version {
		return fmt.Errorf("version mismatch: %s != %s", i.Version, oi.Version)
	}

	return nil
}

func (i *mockInfo) Download(dst string) (string, error) {
	src := filepath.Join("fixtures", "storage", i.Name, i.Version)
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("package name %s with version %s not found", i.Name, i.Version)
		}
		return "", fmt.Errorf("stat %s: %w", src, err)
	}

	target := filepath.Join(dst, "package")
	if err := copy.Copy(src, target); err != nil {
		return "", fmt.Errorf("copy %s to %s: %w", i.Name, target, err)
	}

	return target, nil
}

func (m *mockDownloader) Origin() storage.Origin {
	return &mockInfo{}
}

func (m *mockDownloader) Discover(name string, version string) (storage.Origin, error) {
	return &mockInfo{
		Name:    name,
		Version: version,
	}, nil
}
