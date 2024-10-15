package cti

import (
	"encoding/json"
	"fmt"

	"github.com/acronis/go-raml"
	"github.com/tidwall/gjson"
)

type Entities []*Entity
type EntitiesMap map[string]*Entity

type Entity struct {
	Final             bool                      `json:"final"`
	Cti               string                    `json:"cti"`
	DisplayName       string                    `json:"display_name,omitempty"`
	Description       string                    `json:"description,omitempty"`
	Dictionaries      map[string]interface{}    `json:"dictionaries,omitempty"` // Deprecated
	Values            json.RawMessage           `json:"values,omitempty"`
	Schema            json.RawMessage           `json:"schema,omitempty"`
	TraitsSchema      json.RawMessage           `json:"traits_schema,omitempty"`
	TraitsAnnotations map[GJsonPath]Annotations `json:"traits_annotations,omitempty"`
	Traits            json.RawMessage           `json:"traits,omitempty"`
	Annotations       map[GJsonPath]Annotations `json:"annotations,omitempty"`
	SourceMap         SourceMap                 `json:"source_map,omitempty"`
}

// TODO: This is a temporary structure until proper model is outlined. Used by tests.
type EntityStructured struct {
	Final             bool                      `json:"final"`
	Cti               string                    `json:"cti"`
	DisplayName       string                    `json:"display_name,omitempty"`
	Description       string                    `json:"description,omitempty"`
	Dictionaries      map[string]interface{}    `json:"dictionaries,omitempty"` // Deprecated
	Values            map[string]interface{}    `json:"values,omitempty"`
	Schema            *raml.JSONSchema          `json:"schema,omitempty"`
	TraitsSchema      *raml.JSONSchema          `json:"traits_schema,omitempty"`
	TraitsAnnotations map[GJsonPath]Annotations `json:"traits_annotations,omitempty"`
	Traits            map[string]interface{}    `json:"traits,omitempty"`
	Annotations       map[GJsonPath]Annotations `json:"annotations,omitempty"`
	SourceMap         SourceMap                 `json:"source_map,omitempty"`
}

type Annotations struct {
	Cti           interface{}            `json:"cti.cti,omitempty"` // string or []string
	ID            *bool                  `json:"cti.id,omitempty"`  // string or []string
	DisplayName   *bool                  `json:"cti.display_name,omitempty"`
	Description   *bool                  `json:"cti.description,omitempty"`
	Reference     interface{}            `json:"cti.reference,omitempty"` // bool or string or []string
	Overridable   *bool                  `json:"cti.overridable,omitempty"`
	Final         *bool                  `json:"cti.final,omitempty"`
	Asset         *bool                  `json:"cti.asset,omitempty"`
	L10N          *bool                  `json:"cti.l10n,omitempty"`
	Schema        interface{}            `json:"cti.schema,omitempty"` // string or []string
	Meta          string                 `json:"cti.meta,omitempty"`
	PropertyNames map[string]interface{} `json:"cti.propertyNames,omitempty"`
}

type SourceMap struct {
	TypeAnnotationReference
	InstanceAnnotationReference

	// SourcePath is a relative path to the RAML file where the CTI parent is defined.
	SourcePath string `json:"$sourcePath,omitempty"`

	// OriginalPath is a relative path to RAML fragment where the CTI entity is defined.
	OriginalPath string `json:"$originalPath,omitempty"`
}

func (a *SourceMap) ToBytes() []byte {
	bytes, _ := json.Marshal(a)
	return bytes
}

func (a *SourceMap) HasOriginalPath() bool {
	return a.OriginalPath != ""
}

type AnnotationType struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`

	// Reference is a reference to the annotation type that was used to define the instance.
	Reference string `json:"reference,omitempty"`
}

type TypeAnnotationReference struct {
	Name string `json:"$name,omitempty"`
}

type InstanceAnnotationReference struct {
	AnnotationType *AnnotationType `json:"$annotationType,omitempty"`
}

func (a Annotations) ReadCti() []string {
	if a.Cti == nil {
		return []string{}
	}
	if val, ok := a.Cti.(string); ok {
		return []string{val}
	}
	return a.Cti.([]string)
}

func (a Annotations) ReadReference() string {
	if a.Reference == nil {
		return ""
	}
	if val, ok := a.Reference.(bool); ok {
		return fmt.Sprintf("%t", val)
	}
	return a.Reference.(string)
}

type GJsonPath string

func (k GJsonPath) GetValue(obj []byte) gjson.Result {
	expr := k.String()[1:]
	if expr == "" {
		return gjson.ParseBytes(obj)
	}
	size := len(expr)
	// Trailing ".#" returns a number of elements in an array instead of elements.
	// Keep for reference, but remove when getting the value.
	if expr[size-2:] == ".#" {
		expr = expr[:size-2]
	}
	return gjson.GetBytes(obj, expr)
}

func (k GJsonPath) String() string {
	return string(k)
}
