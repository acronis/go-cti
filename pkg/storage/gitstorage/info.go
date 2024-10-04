package gitstorage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/storage"
)

type gitInfo struct {
	Name string `json:"Name"`
	VCS  string `json:"VCS"`
	URL  string `json:"URL"`
	Hash string `json:"Hash"`
	Ref  string `json:"Ref"`
}

func (i *gitInfo) Validate(o storage.Origin) error {
	oi, ok := o.(*gitInfo)
	if !ok {
		return fmt.Errorf("origin is not a gitInfo")
	}

	if i.VCS != oi.VCS {
		return fmt.Errorf("vcs mismatch: %s != %s", i.VCS, oi.VCS)
	}
	if i.URL != oi.URL {
		return fmt.Errorf("url mismatch: %s != %s", i.URL, oi.URL)
	}
	if i.Hash != oi.Hash {
		return fmt.Errorf("hash mismatch: %s != %s", i.Hash, oi.Hash)
	}
	if i.Ref != oi.Ref {
		return fmt.Errorf("ref mismatch: %s != %s", i.Ref, oi.Ref)
	}

	return nil
}

func (i *gitInfo) Download(cacheDir string) (string, error) {
	filename := fmt.Sprintf("%s-%s-%s.zip", filepath.Base(i.Name), i.Ref, i.Hash[:8])
	cacheZip := filepath.Join(cacheDir, filepath.Dir(i.Name), filename)

	// TODO: download by commit hash not by ref
	if err := gitArchive(i.URL, i.Ref, cacheZip); err != nil {
		return "", err
	}

	destDir := filepath.Join(cacheDir, "package")
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return "", err
	}

	if err := filesys.SecureUnzip(cacheZip, destDir); err != nil {
		return "", fmt.Errorf("unzip %s to %s: %w", cacheZip, destDir, err)
	}

	return destDir, nil
}
