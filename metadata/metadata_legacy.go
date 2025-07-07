package metadata

import (
	"encoding/json"

	"github.com/acronis/go-cti/metadata/consts"
)

type LegacyEntities []LegacyEntity

type LegacyEntity struct {
	Final             bool                       `json:"final"`
	CTI               string                     `json:"cti"`
	Resilient         bool                       `json:"resilient"`
	Access            consts.AccessModifier      `json:"access"`
	DisplayName       string                     `json:"display_name,omitempty"`
	Description       string                     `json:"description,omitempty"`
	Dictionaries      map[string]any             `json:"dictionaries,omitempty"` // Deprecated
	Values            json.RawMessage            `json:"values,omitempty"`
	Schema            json.RawMessage            `json:"schema,omitempty"`
	TraitsSchema      json.RawMessage            `json:"traits_schema,omitempty"`
	TraitsAnnotations map[GJsonPath]*Annotations `json:"traits_annotations,omitempty"`
	Traits            json.RawMessage            `json:"traits,omitempty"`
	Annotations       map[GJsonPath]*Annotations `json:"annotations,omitempty"`
	SourceMap         LegacySourceMap            `json:"source_map,omitempty"`
}

type LegacySourceMap struct {
	TypeAnnotationReference
	InstanceAnnotationReference

	// SourcePath is a relative path to the RAML file where the CTI parent is defined.
	SourcePath string `json:"$sourcePath,omitempty"`

	// OriginalPath is a relative path to RAML fragment where the CTI entity is defined.
	OriginalPath string `json:"$originalPath,omitempty"`
}
