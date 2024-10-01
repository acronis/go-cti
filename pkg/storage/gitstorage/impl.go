package gitstorage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/storage"

	"golang.org/x/mod/semver"
)

type storageImpl struct {
	name string
	ref  string

	location   string
	commitHash string
}

func New() storage.Storage {
	return &storageImpl{}
}

func (g *storageImpl) Origin() storage.Origin {
	return &gitInfo{}
}

func (g *storageImpl) Discover(name string, version string) (storage.DownloadFn, storage.Origin, error) {
	if !semver.IsValid(version) {
		return nil, nil, fmt.Errorf("invalid version %s", version)
	}

	source := fmt.Sprintf("https://%s", name)
	body, err := discoverSource(source)
	if err != nil {
		return nil, nil, fmt.Errorf("discover source at %s: %w", source, err)
	}

	m := goImportRe.FindStringSubmatch(string(body))
	if len(m) == 0 {
		return nil, nil, fmt.Errorf("find go-import at %s", source)
	}
	_, _, sourceLocation := parseGoQuery(m[len(m)-1])
	// TODO: use module.PseudoVersion() to get commit hash
	commitHash, err := gitLsRemote(sourceLocation, version)
	if err != nil {
		return nil, nil, fmt.Errorf("git ls-remote: %w", err)
	}
	if commitHash == "" {
		return nil, nil, fmt.Errorf("failed to find %s %s", sourceLocation, version)
	}

	impl := &storageImpl{
		name:       name,
		ref:        version,
		location:   sourceLocation,
		commitHash: commitHash,
	}
	return impl.download, &gitInfo{
		VCS:  "git",
		URL:  sourceLocation,
		Hash: commitHash,
		Ref:  version,
	}, nil
}

func (g *storageImpl) download(cacheDir string) (string, error) {
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
