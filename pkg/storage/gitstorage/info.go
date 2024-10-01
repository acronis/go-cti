package gitstorage

import (
	"fmt"

	"github.com/acronis/go-cti/pkg/storage"
)

type gitInfo struct {
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
