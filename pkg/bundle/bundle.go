package bundle

import (
	"fmt"
	"os"
	"path/filepath"
)

type Bundle struct {
	Index     *Index
	IndexLock *IndexLock

	BaseDir string
}

// New creates a new bundle from the specified path.
// If the path is empty, the current working directory is used.
func New(path string) (*Bundle, error) {
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get current working directory: %w", err)
		}
		path = cwd
	}
	path = filepath.Join(path, IndexFileName)

	idx, err := ReadIndexFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index file: %w", err)
	}
	idxLock, err := MakeIndexLock(idx.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("make index lock: %w", err)
	}

	return &Bundle{
		Index:     idx,
		IndexLock: idxLock,

		BaseDir: idx.BaseDir,
	}, nil
}

func (b *Bundle) SaveIndexLock() error {
	if err := b.IndexLock.Save(); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}
	return nil
}

func (b *Bundle) SaveIndex() error {
	if err := b.Index.Save(); err != nil {
		return fmt.Errorf("save index: %w", err)
	}
	return nil
}
