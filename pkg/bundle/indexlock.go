package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	IndexLockFileName = "index-lock.json"
	IndexLockVersion  = "v1"
)

type IndexLock struct {
	Version string `json:"version"`
	// Reverse map: key - application code, value - SourceInfo
	Sources map[string]SourceInfo `json:"sources"`
	// Direct map: key - source, value - BundleInfo
	Bundles map[string]Info `json:"bundles"`

	BaseDir  string `json:"-"`
	FilePath string `json:"-"`
}

func (idx *IndexLock) Save() error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index lock: %w", err)
	}

	return os.WriteFile(idx.FilePath, data, 0600)
}

type SourceInfo struct {
	Source string `json:"source"`
}

type Info struct {
	Name      string   `json:"name"`
	AppCode   string   `json:"app_code"`
	Version   string   `json:"version"`
	Integrity string   `json:"integrity"`
	Source    string   `json:"source"`
	Depends   []string `json:"depends"`
}

func MakeIndexLock(pkgDir string) (*IndexLock, error) {
	filePath := filepath.Join(pkgDir, IndexLockFileName)
	idxLock := &IndexLock{
		Version:  IndexLockVersion,
		Sources:  make(map[string]SourceInfo),
		Bundles:  make(map[string]Info),
		BaseDir:  pkgDir,
		FilePath: filePath,
	}

	if data, err := os.ReadFile(filePath); err == nil {
		if err = json.Unmarshal(data, idxLock); err != nil {
			return nil, fmt.Errorf("unmarshal index lock: %w", err)
		}
	}

	return idxLock, nil
}

func ParseIndexDependency(dep string) (string, string) {
	parts := strings.Split(dep, " ")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
