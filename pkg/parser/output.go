package parser

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

type CtiEntities []CtiEntity
type CtiEntitiesMap map[string]*CtiEntity

type CtiEntity struct {
	Final             bool                         `json:"final"`
	Cti               string                       `json:"cti"`
	Dictionaries      json.RawMessage              `json:"dictionaries,omitempty"`
	Values            json.RawMessage              `json:"values,omitempty"`
	Schema            json.RawMessage              `json:"schema,omitempty"`
	TraitsSchema      json.RawMessage              `json:"traits_schema,omitempty"`
	TraitsAnnotations map[GJsonPath]CtiAnnotations `json:"traits_annotations,omitempty"`
	Traits            json.RawMessage              `json:"traits,omitempty"`
	Annotations       map[GJsonPath]CtiAnnotations `json:"annotations,omitempty"`
	SourceMap         SourceMap                    `json:"source_map,omitempty"`
}

type CtiAnnotations struct {
	Cti         interface{} `json:"cti.cti"`       // string or []string
	Id          *bool       `json:"cti.id"`        // string or []string
	Reference   interface{} `json:"cti.reference"` // bool or string or []string
	Overridable *bool       `json:"cti.overridable"`
	Final       *bool       `json:"cti.final"`
	Asset       *bool       `json:"cti.asset"`
	L10N        *bool       `json:"cti.l10n"`
	ReadOnly    *bool       `json:"cti.readOnly"`
	WriteOnly   *bool       `json:"cti.writeOnly"`
	Schema      interface{} `json:"cti.schema"` // string or []string
}

type SourceMap struct {
	TypeAnnotationReference
	InstanceAnnotationReference
	SourcePath   string `json:"$sourcePath"`
	OriginalPath string `json:"$originalPath"`
}

func (a *SourceMap) ToBytes() []byte {
	bytes, _ := json.Marshal(a)
	return bytes
}

func (a *SourceMap) HasOriginalPath() bool {
	return a.OriginalPath != ""
}

type AnnotationType struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Reference string `json:"reference"`
}

type TypeAnnotationReference struct {
	Name string `json:"$name"`
}

type InstanceAnnotationReference struct {
	AnnotationType AnnotationType `json:"$annotationType"`
}

func (a CtiAnnotations) ReadCti() []string {
	if a.Cti == nil {
		return []string{}
	}
	if val, ok := a.Cti.(string); ok {
		return []string{val}
	}
	return a.Cti.([]string)
}

func (a CtiAnnotations) ReadReference() string {
	if a.Reference == nil {
		return ""
	}
	if val, ok := a.Reference.(bool); ok {
		return fmt.Sprintf("%t", val)
	}
	return a.Reference.(string)
}

type GJsonPath string

func (k GJsonPath) GetValue(obj *[]byte) gjson.Result {
	str := k.String()
	expr, size := str[1:], len(str[1:])
	if expr[size-2:] == ".#" {
		expr = expr[:size-2]
	}
	return gjson.GetBytes(*obj, expr)
}

func (k GJsonPath) String() string {
	return string(k)
}
