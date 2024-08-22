package depman

func MakeIndexLock() *IndexLock {
	return &IndexLock{
		Version:  "v1",
		Sources:  map[string]SourceInfo{},
		Packages: map[string]PackageInfo{},
	}
}
