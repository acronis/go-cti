package _package

import (
	"fmt"
)

type Package struct {
	Index     *Index
	IndexLock *IndexLock

	BaseDir string
}

func New(path string) (*Package, error) {
	idx, err := ReadIndexFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}
	idxLock, err := MakeIndexLock(idx.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to make index lock: %w", err)
	}

	return &Package{
		Index:     idx,
		IndexLock: idxLock,

		BaseDir: idx.BaseDir,
	}, nil
}

func (pkg *Package) SaveIndexLock() error {
	if err := pkg.IndexLock.Save(); err != nil {
		return fmt.Errorf("failed to save index lock: %w", err)
	}
	return nil
}

func (pkg *Package) SaveIndex() error {
	if err := pkg.Index.Save(); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}
	return nil
}
