package ctipackage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/metadata/filesys"
)

const (
	IndexLockFileName = "index-lock.json"
	IndexLockVersion  = "v1"
)

type IndexLock struct {
	Version string `json:"version"`
	// Hash of the index.json file
	Hash string `json:"hash,omitempty"`
	// Reverse map: key - package id, value - source
	DependentPackages map[string]string `json:"depends"`
	// Direct map: key - source, value - Info
	SourceInfo map[string]Info `json:"dependsInfo"`
}

func (idx *IndexLock) Save(baseDir string) error {
	return filesys.WriteJSON(filepath.Join(baseDir, IndexLockFileName), idx)
}

type SourceInfo struct {
	Source string `json:"source"`
}

type Info struct {
	PackageID       string            `json:"package_id"`
	Version         string            `json:"version"`
	Integrity       string            `json:"integrity"`
	Source          string            `json:"source"`
	SourceIntegrity string            `json:"source_integrity"`
	Depends         map[string]string `json:"depends"`
}

func ReadIndexLock(pkgDir string) (*IndexLock, error) {
	filePath := filepath.Join(pkgDir, IndexLockFileName)
	idxLock := &IndexLock{
		Version:           IndexLockVersion,
		DependentPackages: make(map[string]string),
		SourceInfo:        make(map[string]Info),
	}

	if err := filesys.ReadJSON(filePath, idxLock); os.IsNotExist(err) {
		return nil, fmt.Errorf("read index lock: %w", err)
	}

	return idxLock, nil
}
