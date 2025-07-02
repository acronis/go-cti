package metadata

import (
	"github.com/acronis/go-raml/v2"
)

type JSONSchemaCTI struct {
	*raml.JSONSchemaGeneric[*JSONSchemaCTI]
	CTI           any            `json:"x-cti.cti,omitempty"` // string or []string
	ID            *bool          `json:"x-cti.id,omitempty"`  // bool?
	Access        AccessModifier `json:"x-cti.access,omitempty"`
	AccessField   *bool          `json:"x-cti.access_field,omitempty"`
	DisplayName   *bool          `json:"x-cti.display_name,omitempty"`
	Description   *bool          `json:"x-cti.description,omitempty"`
	Reference     any            `json:"x-cti.reference,omitempty"` // bool or string or []string
	Overridable   *bool          `json:"x-cti.overridable,omitempty"`
	Final         *bool          `json:"x-cti.final,omitempty"`
	Resilient     *bool          `json:"x-cti.resilient,omitempty"`
	Asset         *bool          `json:"x-cti.asset,omitempty"`
	L10N          *bool          `json:"x-cti.l10n,omitempty"`
	Schema        any            `json:"x-cti.schema,omitempty"` // string or []string
	Meta          string         `json:"x-cti.meta,omitempty"`   // string
	PropertyNames map[string]any `json:"x-cti.propertyNames,omitempty"`
}

func (r *JSONSchemaCTI) Generic() *raml.JSONSchemaGeneric[*JSONSchemaCTI] { return r.JSONSchemaGeneric }

func JSONSchemaWrapper(c *raml.JSONSchemaConverter[*JSONSchemaCTI], core *raml.JSONSchemaGeneric[*JSONSchemaCTI], b *raml.BaseShape) *JSONSchemaCTI {
	w := &JSONSchemaCTI{JSONSchemaGeneric: core}
	if b == nil {
		return w
	}
	if n := b.CustomDomainProperties.Len(); n > 0 {
		for p := b.CustomDomainProperties.Oldest(); p != nil; p = p.Next() {
			val := p.Value.Extension.Value
			switch p.Key {
			case Cti:
				w.CTI = val
			case Final:
				v := val.(bool)
				w.Final = &v
			case Access:
				w.Access = val.(AccessModifier)
			case Resilient:
				v := val.(bool)
				w.Resilient = &v
			case ID:
				v := val.(bool)
				w.ID = &v
			case L10n:
				v := val.(bool)
				w.L10N = &v
			case Asset:
				v := val.(bool)
				w.Asset = &v
			case Overridable:
				v := val.(bool)
				w.Overridable = &v
			case Reference:
				w.Reference = val
			case Schema:
				w.Schema = val
			case Meta:
				w.Meta = val.(string)
			case DisplayName:
				v := val.(bool)
				w.DisplayName = &v
			case Description:
				v := val.(bool)
				w.Description = &v
			case PropertyNames:
				w.PropertyNames = val.(map[string]any)
			}
		}
	}
	// Ignoring custom facets and their values since they are not part of the CTI schema.
	return w
}
