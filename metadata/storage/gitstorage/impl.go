package gitstorage

import (
	"fmt"

	"github.com/acronis/go-cti/metadata/storage"
	"github.com/blang/semver/v4"
)

type storageImpl struct {
}

func New() storage.Storage {
	return &storageImpl{}
}

func (g *storageImpl) Origin() storage.Origin {
	return &gitInfo{}
}

func (g *storageImpl) Discover(name string, version string) (storage.Origin, error) {
	if _, err := semver.Parse(version); err != nil {
		return nil, fmt.Errorf("invalid version %s", version)
	}

	source := fmt.Sprintf("https://%s", name)
	body, err := discoverSource(source)
	if err != nil {
		return nil, fmt.Errorf("discover source at %s: %w", source, err)
	}

	m := goImportRe.FindStringSubmatch(string(body))
	if len(m) == 0 {
		return nil, fmt.Errorf("find go-import at %s", source)
	}
	_, _, sourceLocation := parseGoQuery(m[len(m)-1])
	// TODO: use module.PseudoVersion() to get commit hash
	commitHash, err := gitLsRemote(sourceLocation, version)
	if err != nil {
		return nil, fmt.Errorf("git ls-remote: %w", err)
	}
	if commitHash == "" {
		return nil, fmt.Errorf("failed to find %s %s", sourceLocation, version)
	}

	return &gitInfo{
		VCS:  "git",
		URL:  sourceLocation,
		Hash: commitHash,
		Ref:  version,
	}, nil
}
