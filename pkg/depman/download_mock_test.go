package depman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/storage"

	"github.com/otiai10/copy"
)

type mockDownloader struct {
	source  string
	version string
}

type mockInfo struct {
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

func (m *mockDownloader) Origin() storage.Origin {
	return &mockInfo{}
}

func (m *mockDownloader) Discover(source string, version string) (storage.DownloadFn, storage.Origin, error) {
	m.source = source
	m.version = version

	return m.download, &mockInfo{
		Version: version,
	}, nil
}

func (m *mockDownloader) download(dst string) (string, error) {
	src := filepath.Join("fixtures", "storage", m.source, m.version)
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("bundle source %s with version %s not found", m.source, m.version)
		}
		return "", fmt.Errorf("stat %s: %w", src, err)
	}

	target := filepath.Join(dst, "bundle")
	if err := copy.Copy(src, target); err != nil {
		return "", fmt.Errorf("copy %s to %s: %w", m.source, target, err)
	}

	return target, nil
}
