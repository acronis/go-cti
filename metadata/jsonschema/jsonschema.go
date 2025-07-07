package jsonschema

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-raml/v2"
)

const (
	DefinitionsPrefix = "#/definitions/"
)

type JSONSchemaGeneric = raml.JSONSchemaGeneric[*JSONSchemaCTI]

type JSONSchemaCTI struct {
	JSONSchemaGeneric `yaml:",inline"`
	Annotations       `yaml:",inline"`
}

type Annotations struct {
	CTICTI           any                   `json:"x-cti.cti,omitempty" yaml:"x-cti.cti,omitempty"` // string or []string
	CTIID            *bool                 `json:"x-cti.id,omitempty"  yaml:"x-cti.id,omitempty"`  // bool?
	CTIAccess        consts.AccessModifier `json:"x-cti.access,omitempty" yaml:"x-cti.access,omitempty"`
	CTIAccessField   *bool                 `json:"x-cti.access_field,omitempty" yaml:"x-cti.access_field,omitempty"`
	CTIDisplayName   *bool                 `json:"x-cti.display_name,omitempty" yaml:"x-cti.display_name,omitempty"`
	CTIDescription   *bool                 `json:"x-cti.description,omitempty" yaml:"x-cti.description,omitempty"`
	CTIReference     any                   `json:"x-cti.reference,omitempty" yaml:"x-cti.reference,omitempty"` // bool or string or []string
	CTIOverridable   *bool                 `json:"x-cti.overridable,omitempty" yaml:"x-cti.overridable,omitempty"`
	CTIFinal         *bool                 `json:"x-cti.final,omitempty" yaml:"x-cti.final,omitempty"`
	CTIResilient     *bool                 `json:"x-cti.resilient,omitempty" yaml:"x-cti.resilient,omitempty"`
	CTIAsset         *bool                 `json:"x-cti.asset,omitempty" yaml:"x-cti.asset,omitempty"`
	CTIL10N          *bool                 `json:"x-cti.l10n,omitempty" yaml:"x-cti.l10n,omitempty"`
	CTISchema        any                   `json:"x-cti.schema,omitempty" yaml:"x-cti.schema,omitempty"` // string or []string
	CTIMeta          string                `json:"x-cti.meta,omitempty" yaml:"x-cti.meta,omitempty"`     // string
	CTIPropertyNames map[string]any        `json:"x-cti.propertyNames,omitempty" yaml:"x-cti.propertyNames,omitempty"`
}

func (a *Annotations) IsEmpty() bool {
	return a == nil || (a.CTICTI == nil && a.CTIID == nil && a.CTIAccess == "" &&
		a.CTIAccessField == nil && a.CTIDisplayName == nil && a.CTIDescription == nil &&
		a.CTIReference == nil && a.CTIOverridable == nil && a.CTIFinal == nil &&
		a.CTIResilient == nil && a.CTIAsset == nil && a.CTIL10N == nil &&
		a.CTISchema == nil && a.CTIMeta == "" && len(a.CTIPropertyNames) == 0)
}

func (r *JSONSchemaCTI) Generic() *JSONSchemaGeneric { return &r.JSONSchemaGeneric }

func JSONSchemaWrapper(c *raml.JSONSchemaConverter[*JSONSchemaCTI], core *JSONSchemaGeneric, b *raml.BaseShape) *JSONSchemaCTI {
	if core == nil {
		return nil
	}
	w := &JSONSchemaCTI{JSONSchemaGeneric: *core}
	if b == nil {
		return w
	}
	for p := b.CustomDomainProperties.Oldest(); p != nil; p = p.Next() {
		val := p.Value.Extension.Value
		switch p.Key {
		case consts.CTI:
			w.CTICTI = val
		case consts.Final:
			v := val.(bool)
			w.CTIFinal = &v
		case consts.Access:
			w.CTIAccess = val.(consts.AccessModifier)
		case consts.Resilient:
			v := val.(bool)
			w.CTIResilient = &v
		case consts.ID:
			v := val.(bool)
			w.CTIID = &v
		case consts.L10n:
			v := val.(bool)
			w.CTIL10N = &v
		case consts.Asset:
			v := val.(bool)
			w.CTIAsset = &v
		case consts.Overridable:
			v := val.(bool)
			w.CTIOverridable = &v
		case consts.Reference:
			w.CTIReference = val
		case consts.Schema:
			w.CTISchema = val
		case consts.Meta:
			w.CTIMeta = val.(string)
		case consts.DisplayName:
			v := val.(bool)
			w.CTIDisplayName = &v
		case consts.Description:
			v := val.(bool)
			w.CTIDescription = &v
		case consts.PropertyNames:
			w.CTIPropertyNames = val.(map[string]any)
		}
	}
	// Ignoring custom facets and their values since they are not part of the CTI schema.
	return w
}

// GetRefType extracts the type from a ref value.
// E.g.: "MarketingInfo" from "#/definitions/MarketingInfo"
func GetRefType(ref string) (string, error) {
	if strings.HasPrefix(ref, DefinitionsPrefix) {
		return ref[len(DefinitionsPrefix):], nil
	}
	return "", errors.New("non-definition references are not implemented")
}

func (js *JSONSchemaCTI) Map() map[string]any {
	if js == nil {
		return nil
	}
	m := js.JSONSchemaGeneric.Map()
	if js.CTICTI != nil {
		m[consts.CTI] = js.CTICTI
	}
	if js.CTIID != nil {
		m[consts.ID] = *js.CTIID
	}
	if js.CTIAccess != "" {
		m[consts.Access] = js.CTIAccess
	}
	if js.CTIAccessField != nil {
		m[consts.AccessField] = *js.CTIAccessField
	}
	if js.CTIDisplayName != nil {
		m[consts.DisplayName] = *js.CTIDisplayName
	}
	if js.CTIDescription != nil {
		m[consts.Description] = *js.CTIDescription
	}
	if js.CTIReference != nil {
		m[consts.Reference] = js.CTIReference
	}
	if js.CTIOverridable != nil {
		m[consts.Overridable] = *js.CTIOverridable
	}
	if js.CTIFinal != nil {
		m[consts.Final] = *js.CTIFinal
	}
	if js.CTIResilient != nil {
		m[consts.Resilient] = *js.CTIResilient
	}
	if js.CTIAsset != nil {
		m[consts.Asset] = *js.CTIAsset
	}
	if js.CTIL10N != nil {
		m[consts.L10n] = *js.CTIL10N
	}
	if js.CTISchema != nil {
		m[consts.Schema] = js.CTISchema
	}
	if js.CTIMeta != "" {
		m[consts.Meta] = js.CTIMeta
	}
	if len(js.CTIPropertyNames) > 0 {
		m[consts.PropertyNames] = js.CTIPropertyNames
	}
	return m
}

func (js *JSONSchemaCTI) ShallowCopy() *JSONSchemaCTI {
	if js == nil {
		return nil
	}
	// Create a new instance of JSONSchemaCTI and copy the fields.
	newJS := &JSONSchemaCTI{}
	newJS.JSONSchemaGeneric = *js.JSONSchemaGeneric.ShallowCopy()
	newJS.Annotations = js.Annotations
	return newJS
}

func (js *JSONSchemaCTI) DeepCopy() *JSONSchemaCTI {
	if js == nil {
		return nil
	}
	newJS := &JSONSchemaCTI{}
	newJS.JSONSchemaGeneric = *js.JSONSchemaGeneric.DeepCopy()
	newJS.Annotations = js.Annotations
	if len(js.CTIPropertyNames) > 0 {
		newJS.CTIPropertyNames = make(map[string]any, len(js.CTIPropertyNames))
		for k, v := range js.CTIPropertyNames {
			newJS.CTIPropertyNames[k] = v
		}
	}
	return newJS
}

func (js *JSONSchemaCTI) IsCompatibleWith(schema *JSONSchemaCTI) bool {
	// If schema is an "any" type, is "ref", or either of types is "anyOf", assume compatibility.
	return js != nil && (schema.IsAny() || schema.IsAnyOf() || js.IsAnyOf() || js.IsRef() || js.Type == schema.Type)
}

func (js *JSONSchemaCTI) GetRefSchema() (*JSONSchemaCTI, string, error) {
	if js == nil {
		return nil, "", errors.New("invalid schema")
	}
	if js.Definitions == nil {
		return nil, "", errors.New("schema has no definitions")
	}
	typeName, err := GetRefType(js.Ref)
	if err != nil {
		return nil, "", fmt.Errorf("get ref type: %w", err)
	}
	schema, ok := js.Definitions[typeName]
	if !ok {
		return nil, "", fmt.Errorf("schema does not have ref: %s", typeName)
	}
	return schema, typeName, nil
}

func (js *JSONSchemaCTI) IsRef() bool {
	return js != nil && js.Ref != ""
}

func (js *JSONSchemaCTI) IsAnyOf() bool {
	return js != nil && js.AnyOf != nil && js.Type == ""
}

func (js *JSONSchemaCTI) IsAny() bool {
	// An "any" type is one that has no type defined and is not an anyOf.
	return js != nil && js.AnyOf == nil && js.Type == ""
}
