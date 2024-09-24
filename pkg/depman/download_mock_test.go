package depman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/downloader"

	"github.com/otiai10/copy"
)

type mockDownloader struct {
	source  string
	version string
}

func (m *mockDownloader) Discover(source string, version string) (downloader.DownloadFn, downloader.Info, error) {
	m.source = source
	m.version = version

	return m.download, downloader.Info{
		VCS:  "mock",
		URL:  "mock:" + source + "@" + version,
		Hash: "mock-hash",
		Ref:  version,
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
