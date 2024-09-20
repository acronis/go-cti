package bundle

import (
	"fmt"
)

type Bundle struct {
	Index     *Index
	IndexLock *IndexLock

	BaseDir string
}

func New(path string) (*Bundle, error) {
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
