package godownloader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/downloader"
	"github.com/acronis/go-cti/pkg/filesys"
)

type downloaderImpl struct {
	name string
	ref  string

	location   string
	commitHash string
}

func New() downloader.Downloader {
	return &downloaderImpl{}
}

func (g *downloaderImpl) Discover(name string, version string) (downloader.DownloadFn, downloader.Info, error) {
	source := fmt.Sprintf("https://%s", name)
	body, err := discoverSource(source)
	if err != nil {
		return nil, downloader.Info{}, fmt.Errorf("discover source at %s: %w", source, err)
	}

	m := goImportRe.FindStringSubmatch(string(body))
	if len(m) == 0 {
		return nil, downloader.Info{}, fmt.Errorf("find go-import at %s", source)
	}
	_, _, sourceLocation := parseGoQuery(m[len(m)-1])

	commitHash, err := gitLsRemote(sourceLocation, version)
	if err != nil {
		return nil, downloader.Info{}, fmt.Errorf("git ls-remote: %w", err)
	}
	if commitHash == "" {
		return nil, downloader.Info{}, fmt.Errorf("failed to find %s %s", sourceLocation, version)
	}

	impl := &downloaderImpl{
		name:       name,
		ref:        version,
		location:   sourceLocation,
		commitHash: commitHash,
	}
	return impl.download, downloader.Info{
		VCS:  "git",
		URL:  sourceLocation,
		Hash: commitHash,
		Ref:  version,
	}, nil
}

func (g *downloaderImpl) download(cacheDir string) (string, error) {
	filename := fmt.Sprintf("%s-%s-%s.zip", filepath.Base(g.name), g.ref, g.commitHash[:8])
	cacheZip := filepath.Join(cacheDir, filepath.Dir(g.name), filename)

	// TODO: download by commit hash not by ref
	if err := gitArchive(g.location, g.ref, cacheZip); err != nil {
		return "", err
	}

	destDir := filepath.Join(cacheDir, "bundle")
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return "", err
	}

	if err := filesys.SecureUnzip(cacheZip, destDir); err != nil {
		return "", fmt.Errorf("unzip %s to %s: %w", cacheZip, destDir, err)
	}

	return destDir, nil
}
