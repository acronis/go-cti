package parser

type Bundle struct {
	Assets   map[string]string `json:"assets,omitempty"`
	Entities CtiEntities       `json:"entities"`
}
