package packager

type AdditionalInfo struct {
	Types      []string `json:"types,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type IndexJSON struct {
	Type           string          `json:"type"`
	Entities       []string        `json:"entities"`
	Schema         string          `json:"$schema,omitempty"`
	Apis           []string        `json:"apis,omitempty"`
	Examples       []string        `json:"example,omitempty"`
	Assets         []string        `json:"assets,omitempty"`
	Dictionaries   []string        `json:"dictionaries,omitempty"`
	AdditionalInfo *AdditionalInfo `json:"additional_info,omitempty"`
}
