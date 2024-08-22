package depman

type IndexLock struct {
	Version string `json:"version"`
	// Reverse map: key - application code, value - SourceInfo
	Sources map[string]SourceInfo `json:"sources"`
	// Direct map: key - source, value - PackageInfo
	Packages map[string]PackageInfo `json:"packages"`
}

type SourceInfo struct {
	Source string `json:"source"`
}

type PackageInfo struct {
	Name      string   `json:"name"`
	AppCode   string   `json:"app_code"`
	Version   string   `json:"version"`
	Integrity string   `json:"integrity"`
	Source    string   `json:"source"`
	Depends   []string `json:"depends"`
}
