package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata/attribute_selector"
	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"
)

type Entities []Entity

// Sort sorts the entities by their CTI identifiers.
func (items Entities) Sort() {
	sort.Slice(items, func(a, b int) bool {
		return items[a].GetCTI() < items[b].GetCTI()
	})
}

type EntityTypeMap map[string]*EntityType
type EntityInstanceMap map[string]*EntityInstance
type EntityMap map[string]Entity

// TODO: For future use. Need to create a context and move all creation methods there.
type MContext struct{}

// PtrReplacer is an interface for objects that can replace their pointers with another object of the same type.
type PtrReplacer[T any] interface {
	// ReplacePointer replaces the pointer receiver with the provided object.
	ReplacePointer(T) error
}

type NilChecker interface {
	// IsNil checks if the underlying pointer receiver is nil.
	IsNil() bool
}

// Entity is an interface that represents a CTI entity.
// It provides methods to access and manipulate the entity's properties, annotations, and relationships.
// See more information about CTI entities in the [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#cti-and-metadata
type Entity interface {
	// GetCTI returns the CTI identifier of the entity as a string.
	GetCTI() string

	// SetFinal sets the final flag for the entity.
	SetFinal(final bool)
	// IsFinal returns true if the entity is final, meaning it cannot be extended or modified.
	// Derived EntityType entities are not allowed to have final entity as parent.
	IsFinal() bool

	// GetAccess returns the access modifier of the entity.
	// Access modifiers are used to control visibility and accessibility of the entity.
	GetAccess() consts.AccessModifier
	// SetAccess sets the access modifier for the entity.
	SetAccess(access consts.AccessModifier)
	// IsSamePackage checks if the entity belongs to the same package as the other entity.
	IsSamePackage(other Entity) bool
	// IsSameVendor checks if the entity belongs to the same vendor as the other entity.
	IsSameVendor(other Entity) bool
	// IsAccessibleBy checks if the entity is accessible by the other entity based on its access modifier.
	IsAccessibleBy(other Entity) error

	// IsA checks if the entity is a subtype of the given EntityType.
	IsA(other *EntityType) bool

	// IsChildOf checks if the entity is a direct child of the given EntityType.
	IsChildOf(other *EntityType) bool

	// SetResilient sets the resilient flag for the entity.
	SetResilient(resilient bool)
	// SetDisplayName sets the display name for the entity.
	SetDisplayName(displayName string)
	// SetDescription sets the description for the entity.
	SetDescription(description string)
	// SetDictionaries sets the dictionaries for the entity.
	SetDictionaries(dictionaries map[string]any)

	// Parent returns the parent EntityType of the entity, if any.
	// Parent can be only EntityType.
	Parent() *EntityType
	// SetParent sets the parent EntityType for the entity.
	SetParent(*EntityType) error

	// Expression returns the parsed CTI expression of the entity.
	// The method is lazy initializer. If the expression is not parsed yet, it will be parsed on the first call.
	Expression() (*cti.Expression, error)
	// Match checks if the entity matches the other entity based on their CTI expressions.
	Match(other Entity) (bool, error)
	// Vendor returns the entity vendor from CTI expression.
	// If expression fails to parse, it returns empty string.
	Vendor() string
	// Package returns the entity package from CTI expression.
	// If expression fails to parse, it returns empty string.
	Package() string
	// Name returns the entity name from CTI expression.
	// If expression fails to parse, it returns empty string.
	Name() string
	// Version returns the entity version from CTI expression.
	// If expression fails to parse, it returns an empty Version object.
	Version() cti.Version

	Context() *MContext

	// GetAnnotations returns the annotations of the entity.
	// Annotations are used to store additional metadata about the entity.
	GetAnnotations() map[GJsonPath]*Annotations
	// FindAnnotationsByPredicateInChain finds annotations by key in the entity and its parent chain based
	// on the provided predicate function.
	// If the key is not found, it returns nil.
	FindAnnotationsByPredicateInChain(key GJsonPath, predicate func(*Annotations) bool) *Annotations
	// FindAnnotationsByKeyInChain finds annotations by key in the entity and its parent chain.
	// If the key is not found, it returns nil.
	FindAnnotationsByKeyInChain(key GJsonPath) *Annotations

	NilChecker
}

// Annotations represents a set of annotations for a CTI entity.
// For more information, see [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#cti-type-extensions
type Annotations struct {
	CTI           any                   `json:"cti.cti,omitempty" yaml:"cti.cti,omitempty"` // string or []string
	ID            *bool                 `json:"cti.id,omitempty" yaml:"cti.id,omitempty"`
	Access        consts.AccessModifier `json:"cti.access,omitempty" yaml:"cti.access,omitempty"`
	AccessField   *bool                 `json:"cti.access_field,omitempty" yaml:"cti.access_field,omitempty"`
	DisplayName   *bool                 `json:"cti.display_name,omitempty" yaml:"cti.display_name,omitempty"`
	Description   *bool                 `json:"cti.description,omitempty" yaml:"cti.description,omitempty"`
	Reference     any                   `json:"cti.reference,omitempty" yaml:"cti.reference,omitempty"` // bool or string or []string
	Overridable   *bool                 `json:"cti.overridable,omitempty" yaml:"cti.overridable,omitempty"`
	Final         *bool                 `json:"cti.final,omitempty" yaml:"cti.final,omitempty"`
	Resilient     *bool                 `json:"cti.resilient,omitempty" yaml:"cti.resilient,omitempty"`
	Asset         *bool                 `json:"cti.asset,omitempty" yaml:"cti.asset,omitempty"`
	L10N          *bool                 `json:"cti.l10n,omitempty" yaml:"cti.l10n,omitempty"`
	Schema        any                   `json:"cti.schema,omitempty" yaml:"cti.schema,omitempty"` // string or []string
	Meta          string                `json:"cti.meta,omitempty" yaml:"cti.meta,omitempty"`     // string
	PropertyNames map[string]any        `json:"cti.propertyNames,omitempty" yaml:"cti.propertyNames,omitempty"`
}

type AnnotationType struct {
	// Name is the name of the annotation type.
	Name string `json:"name,omitempty"`
	// Type is the type of the annotation. Can be either "object" or "array".
	Type string `json:"type,omitempty"`

	// Reference is a reference to the annotation type that was used to define the instance.
	Reference string `json:"reference,omitempty"`
}

// ReadCTI returns a slice of CTI identifiers.
// If the CTI annotation is a string, it returns a slice with that string.
// If it is a slice, it returns a slice with all strings from the slice.
// If the CTI annotation is nil, it returns an empty slice.
// If the Schema annotation is not a string or a slice, it returns an empty slice.
func (a *Annotations) ReadCTI() []string {
	if a == nil || a.CTI == nil {
		return []string{}
	}
	if val, ok := a.CTI.(string); ok {
		return []string{val}
	}
	var vals []string
	for _, val := range a.CTI.([]any) {
		if strVal, ok := val.(string); ok {
			vals = append(vals, strVal)
		}
	}
	return vals
}

// ReadSchema returns a slice of CTI schema identifiers.
// If the Schema annotation is a string, it returns a slice with that string.
// If it is a slice, it returns a slice with all strings from the slice.
// If the Schema annotation is nil, it returns an empty slice.
// If the Schema annotation is not a string or a slice, it returns an empty slice.
func (a *Annotations) ReadCTISchema() []string {
	if a == nil || a.Schema == nil {
		return []string{}
	}
	if val, ok := a.Schema.(string); ok {
		return []string{val}
	}
	var vals []string
	for _, val := range a.Schema.([]any) {
		switch v := val.(type) {
		case string:
			vals = append(vals, v)
		case nil:
			vals = append(vals, "null")
		}
	}
	return vals
}

// ReadReference returns a slice of CTI reference values.
// If the Reference annotation is a boolean, it returns a slice with that boolean as a string.
// If it is a string, it returns a slice with that string.
// If it is a slice, it returns a slice with all strings from the slice.
// If the Reference annotation is not a boolean, string, or slice, it returns an empty slice.
func (a *Annotations) ReadReference() []string {
	if a == nil {
		return []string{}
	}
	switch t := a.Reference.(type) {
	case bool:
		return []string{strconv.FormatBool(t)}
	case string:
		return []string{t}
	case []any:
		var vals []string
		for _, val := range t {
			if strVal, ok := val.(string); ok {
				vals = append(vals, strVal)
			}
		}
		return vals
	}
	return nil
}

// GJsonPath is a type that represents a [GJSON](https://github.com/tidwall/gjson) path.
type GJsonPath string

// GetValue returns the value of the GJsonPath from the given JSON bytes.
// If the path is empty, it returns the entire JSON object.
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

// String returns the GJsonPath as a string.
func (k GJsonPath) String() string {
	return string(k)
}

// Base properties for all CTI entities.
// Provides a common implementation for Entity interface. Some methods are not implemented
// and are overridden by EntityType and EntityInstance structs.
// The structure is defined according to the [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#metadata-structure
type entity struct {
	// Final indicates that the entity is final and cannot be extended or modified.
	Final bool `json:"final" yaml:"final"`

	// Access indicates the access modifier of the entity.
	Access consts.AccessModifier `json:"access" yaml:"access"`

	// CTI is the CTI identifier of the entity.
	CTI string `json:"cti" yaml:"cti"`

	// Resilient indicates that the entity is resilient and can be used in resilient contexts.
	Resilient bool `json:"resilient" yaml:"resilient"`

	// DisplayName is the display name of the entity.
	DisplayName string `json:"display_name,omitempty" yaml:"display_name,omitempty"`

	// Description is the description of the entity.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Dictionaries is a map of dictionaries for the entity.
	Dictionaries map[string]any `json:"dictionaries,omitempty" yaml:"dictionaries,omitempty"`

	// Annotations is a map of annotations for the entity.
	Annotations map[GJsonPath]*Annotations `json:"annotations,omitempty" yaml:"annotations,omitempty"`

	// parent is the parent entity type of the entity.
	parent *EntityType `json:"-" yaml:"-"` // Parent entity type, if any

	// expression is the parsed CTI expression of the entity.
	expression *cti.Expression `json:"-" yaml:"-"` // Parsed CTI expression, if any

	// ctx is current unused.
	ctx *MContext `json:"-" yaml:"-"`
}

type DocumentSourceMap struct {
	// SourcePath is a relative path to the source file where the CTI parent is defined.
	SourcePath string `json:"$sourcePath,omitempty" yaml:"$sourcePath,omitempty"`

	// OriginalPath is a relative path to source fragment where the CTI entity is defined.
	OriginalPath string `json:"$originalPath,omitempty" yaml:"$originalPath,omitempty"`

	// Line is the line number in the source file where the CTI entity is defined.
	Line int `json:"$line,omitempty" yaml:"$line,omitempty"`
}

// GetCTI returns the CTI identifier of the entity as a string.
func (e *entity) GetCTI() string {
	return e.CTI
}

// GetAccess returns the access modifier of the entity.
func (e *entity) GetAccess() consts.AccessModifier {
	return e.Access
}

// Parent returns the parent EntityType of the entity, if any.
func (e *entity) Parent() *EntityType {
	return e.parent
}

// GetAnnotations returns the annotations of the entity.
func (e *entity) GetAnnotations() map[GJsonPath]*Annotations {
	return e.Annotations
}

// FindAnnotationsByPredicateInChain finds annotations by key in the entity and its parent chain
// based on the provided predicate function.
// If the key is not found, it returns nil.
func (e *entity) FindAnnotationsByPredicateInChain(key GJsonPath, predicate func(*Annotations) bool) *Annotations {
	var root Entity = e
	for !root.IsNil() {
		annotations := root.GetAnnotations()
		if val, ok := annotations[key]; ok && predicate(val) {
			return val
		}
		root = root.Parent()
	}
	return nil
}

// FindAnnotationsByKeyInChain finds annotations by key in the entity and its parent chain.
// If the key is not found, it returns nil.
func (e *entity) FindAnnotationsByKeyInChain(key GJsonPath) *Annotations {
	var root Entity = e
	for !root.IsNil() {
		annotations := root.GetAnnotations()
		if val, ok := annotations[key]; ok {
			return val
		}
		root = root.Parent()
	}
	return nil
}

// Context returns the current context of the entity. Current unused.
func (e *entity) Context() *MContext {
	return e.ctx
}

// IsAccessibleBy checks if the entity is accessible by the other entity based on its access modifier.
func (e *entity) IsAccessibleBy(other Entity) error {
	if other == nil {
		return errors.New("other entity is nil")
	}
	if !e.IsSameVendor(other) && e.Access != consts.AccessModifierPublic {
		return errors.New("cannot reference non-public entity of external vendor")
	} else if !e.IsSamePackage(other) && e.Access == consts.AccessModifierPrivate {
		return errors.New("cannot reference private entity of the same vendor")
	}
	return nil
}

// IsSameVendor checks if the entity belongs to the same vendor as the other entity.
func (e *entity) IsSameVendor(other Entity) bool {
	if other == nil {
		return false
	}
	if e.Vendor() != other.Vendor() {
		return false
	}
	return true
}

// IsSamePackage checks if the entity belongs to the same package as the other entity.
func (e *entity) IsSamePackage(other Entity) bool {
	if other == nil {
		return false
	}
	if e.Vendor() != other.Vendor() {
		return false
	}
	if e.Package() != other.Package() {
		return false
	}
	return true
}

// Vendor returns the entity vendor from CTI expression.
func (e *entity) Vendor() string {
	expr, err := e.Expression()
	if err != nil {
		return ""
	}
	tail := expr.Tail()
	return string(tail.Vendor)
}

// Package returns the entity package from CTI expression.
func (e *entity) Package() string {
	expr, err := e.Expression()
	if err != nil {
		return ""
	}
	tail := expr.Tail()
	return string(tail.Package)
}

// Name returns the entity name from CTI expression.
func (e *entity) Name() string {
	expr, err := e.Expression()
	if err != nil {
		return ""
	}
	tail := expr.Tail()
	return string(tail.EntityName)
}

// Version returns the entity version from CTI expression.
func (e *entity) Version() cti.Version {
	expr, err := e.Expression()
	if err != nil {
		return cti.Version{}
	}
	tail := expr.Tail()
	return tail.Version
}

// SetParent sets the parent EntityType for the entity. Not implemented in the base entity type.
func (e *entity) SetParent(_ *EntityType) error {
	return errors.New("entity does not implement SetParent")
}

// ReplacePointer replaces the pointer receiver with the provided object. Not implemented in the base entity type.
func (e *entity) ReplacePointer(_ Entity) error {
	return errors.New("entity does not implement ReplacePointer")
}

// IsFinal returns true if the entity is final, meaning it cannot be extended.
func (e *entity) IsFinal() bool {
	return e.Final
}

// Expression returns the parsed CTI expression of the entity.
func (e *entity) Expression() (*cti.Expression, error) {
	if e.expression == nil {
		if e.CTI == "" {
			return nil, errors.New("entity CTI is empty")
		}
		expr, err := cti.ParseIdentifier(e.CTI)
		if err != nil {
			return nil, fmt.Errorf("parse expression %s: %w", e.CTI, err)
		}
		e.expression = &expr
	}
	return e.expression, nil
}

// IsA checks if the entity is a subtype of the given EntityType.
func (e *entity) IsA(typ *EntityType) bool {
	if typ == nil {
		return false
	}
	return strings.HasPrefix(e.CTI, typ.CTI)
}

// IsChildOf checks if the entity is a direct child of the given EntityType.
func (e *entity) IsChildOf(parent *EntityType) bool {
	if parent == nil {
		return false
	}
	return GetParentCTI(e.CTI) == parent.CTI
}

// Match checks if the entity matches the other entity based on their CTI expressions.
func (e *entity) Match(other Entity) (bool, error) {
	if other == nil {
		return false, errors.New("other entity is nil")
	}
	expr, err := e.Expression()
	if err != nil {
		return false, fmt.Errorf("get entity expression: %w", err)
	}
	otherExpr, err := other.Expression()
	if err != nil {
		return false, fmt.Errorf("get other entity expression: %w", err)
	}
	ok, err := expr.MatchIgnoreQuery(*otherExpr)
	if err != nil {
		return false, fmt.Errorf("failed to match expression: %w", err)
	} else if !ok {
		return false, fmt.Errorf("expression %s does not match %s", expr, otherExpr)
	}
	return true, nil
}

// SetFinal sets the final flag for the entity.
func (e *entity) SetFinal(final bool) {
	e.Final = final
}

// SetAccess sets the access modifier for the entity.
func (e *entity) SetAccess(access consts.AccessModifier) {
	e.Access = access
}

// SetResilient sets the resilient flag for the entity.
func (e *entity) SetResilient(resilient bool) {
	e.Resilient = resilient
}

// SetDisplayName sets the display name for the entity.
func (e *entity) SetDisplayName(displayName string) {
	e.DisplayName = displayName
}

// SetDescription sets the description for the entity.
func (e *entity) SetDescription(description string) {
	e.Description = description
}

// SetDictionaries sets the dictionaries for the entity.
func (e *entity) SetDictionaries(dictionaries map[string]any) {
	e.Dictionaries = dictionaries
}

// SetAnnotations sets the annotations for the entity.
func (e *entity) SetAnnotations(annotations map[GJsonPath]*Annotations) {
	e.Annotations = annotations
}

// IsNil checks if the underlying pointer receiver is nil.
func (e *entity) IsNil() bool {
	return e == nil
}

// NewEntityType creates a new EntityType with the given CTI identifier, schema, and annotations.
func NewEntityType(
	id string,
	schema *jsonschema.JSONSchemaCTI,
	annotations map[GJsonPath]*Annotations,
) (*EntityType, error) {
	if id == "" {
		return nil, errors.New("identifier is empty")
	}
	if schema == nil {
		return nil, errors.New("schema is nil")
	}
	if annotations == nil {
		annotations = make(map[GJsonPath]*Annotations)
	}

	obj := &EntityType{
		entity: entity{
			CTI:         id,
			Final:       true, // All entities are final by default
			Access:      consts.AccessModifierProtected,
			Annotations: annotations,
		},
		Schema: schema,
	}

	return obj, nil
}

// EntityType represents a CTI type which is a domain type with data schema and optional trait schema and traits.
// It is used to define the contract between the domain and the implementation.
// For more information, see [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#data-types-and-traits
type EntityType struct {
	entity `yaml:",inline"`

	// Schema is the JSON schema of the entity type. Must be present.
	Schema *jsonschema.JSONSchemaCTI `json:"schema" yaml:"schema"`

	// TraitsSchema is the JSON schema of the traits for the entity type. Optional.
	TraitsSchema *jsonschema.JSONSchemaCTI `json:"traits_schema,omitempty" yaml:"traits_schema,omitempty"`

	// TraitsAnnotations is a map of annotations for the traits schema. Must be present if TraitsSchema is set.
	TraitsAnnotations map[GJsonPath]*Annotations `json:"traits_annotations,omitempty" yaml:"traits_annotations,omitempty"`

	// TraitsSourceMap is the information about the source of traits schema.
	TraitsSourceMap *TypeSourceMap `json:"traits_source_map,omitempty" yaml:"traits_source_map,omitempty"`

	// Traits is a map of traits for the entity type.
	// Optional, may be present if parent entity type defines traits schema.
	Traits map[string]any `json:"traits,omitempty" yaml:"traits,omitempty"`

	// mergedSchema is a cached merged schema of the entity type and its parent chain.
	mergedSchema *jsonschema.JSONSchemaCTI `json:"-" yaml:"-"`

	// validatorSchema is a cached schema compiled with gojsonschema
	validatorSchema *gojsonschema.Schema

	// validatorTraitsSchema is a cached traits schema compiled with gojsonschema
	validatorTraitsSchema *gojsonschema.Schema

	// mergedTraits is a cached map of merged traits from the entity type and its parent chain.
	mergedTraits map[string]any `json:"-" yaml:"-"`

	// rawSchema is the cached marshaled value of Schema.
	rawSchema []byte `json:"-" yaml:"-"`

	// rawTraitValues is the cached marshaled value of Traits.
	rawTraitValues []byte `json:"-" yaml:"-"`

	// SourceMap is the information about the source of the entity.
	SourceMap *TypeSourceMap `json:"source_map,omitempty" yaml:"source_map,omitempty"`
}

type TypeSourceMap struct {
	// Name is the name of the entity type as it was defined in the source.
	Name string `json:"$name,omitempty" yaml:"$name,omitempty"`

	DocumentSourceMap `yaml:",inline"`
}

// SetParent sets the parent EntityType for the entity.
// If provided parent entity is nil, it removes the parent reference.
// If provided parent entity is final, it returns an error.
func (e *EntityType) SetParent(entity *EntityType) error {
	if entity == nil {
		e.parent = nil
		return nil
	}
	if entity.IsFinal() {
		return errors.New("cannot set parent to a final type")
	}
	e.parent = entity
	return nil
}

// GetMergedSchema returns the merged schema of the entity type and its parent chain.
//
// Method provides lazy initialization of the merged schema and caches the result.
// Use ResetMergedSchema to reset the cached schema if necessary.
func (e *EntityType) GetMergedSchema() (*jsonschema.JSONSchemaCTI, error) {
	if e.Schema == nil {
		return nil, errors.New("entity type schema is nil")
	} else if e.parent == nil {
		return e.Schema, nil
	} else if e.mergedSchema != nil {
		return e.mergedSchema, nil
	}

	// Copy the child schema since it will be modified during the merge process.
	childRootSchema := e.Schema.DeepCopy()

	childSchema, refType, err := childRootSchema.GetRefSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema definition: %w", err)
	}

	definitions := map[string]*jsonschema.JSONSchemaCTI{}
	for k, v := range childRootSchema.Definitions {
		if k == refType {
			continue
		}
		definitions[k] = v
	}

	origSelfRefType := "#/definitions/" + refType
	refsToReplace := map[string]struct{}{}

	parent := e.Parent()
	for parent != nil {
		// Copy the parent schema since it may be modified during the merge process.
		parentRootSchema := parent.Schema.DeepCopy()

		parentSchema, parentRefType, err := parentRootSchema.GetRefSchema()
		if err != nil {
			return nil, fmt.Errorf("failed to extract parent schema definition: %w", err)
		}
		refsToReplace["#/definitions/"+parentRefType] = struct{}{}

		childSchema, err = jsonschema.MergeSchemas(parentSchema, childSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to merge schemas: %w", err)
		}

		for parentDefName, parentDef := range parentRootSchema.Definitions {
			if parentDefName == parentRefType {
				continue
			}
			if childDef, ok := definitions[parentDefName]; ok {
				childDef, err = jsonschema.MergeSchemas(parentDef, childDef)
				if err != nil {
					return nil, fmt.Errorf("failed to merge definitions: %w", err)
				}
				definitions[parentDefName] = childDef
			} else {
				definitions[parentDefName] = parentDef
			}
		}
		parent = parent.Parent()
	}
	definitions[refType] = childSchema
	for _, definition := range definitions {
		if err = jsonschema.FixSelfReferences(definition, origSelfRefType, refsToReplace); err != nil {
			return nil, fmt.Errorf("failed to fix self references: %w", err)
		}
	}

	e.mergedSchema = &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Version:     "http://json-schema.org/draft-07/schema",
			Ref:         origSelfRefType,
			Definitions: definitions,
		},
	}

	return e.mergedSchema, nil
}

// ResetMergedSchema reset the cached merged schema of the entity type.
// This is useful when Schema has been modified and needs to be reloaded.
func (e *EntityType) ResetMergedSchema() {
	e.mergedSchema = nil
	e.validatorSchema = nil
}

// GetTraitsSchema returns the traits schema of the entity type.
// If the entity type does not have a traits schema, it returns nil.
func (e *EntityType) GetTraitsSchema() *jsonschema.JSONSchemaCTI {
	return e.TraitsSchema
}

// GetSchemaByAttributeSelectorInChain returns the sub-schema by the given attribute selector in the entity type
// and its parent chain.
// For more information, see [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#attribute-selector
func (e *EntityType) GetSchemaByAttributeSelectorInChain(attributeSelector string) (*jsonschema.JSONSchemaCTI, error) {
	as, err := attribute_selector.NewAttributeSelector(attributeSelector)
	if err != nil {
		return nil, fmt.Errorf("create attribute selector: %w", err)
	}
	// Use merged schema to ensure that we can get any property in the chain.
	schema, err := e.GetMergedSchema()
	if err != nil {
		return nil, fmt.Errorf("get merged schema: %w", err)
	}
	schema, _, err = schema.GetRefSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema definition: %w", err)
	}
	return as.WalkJSONSchema(schema)
}

// FindEntityTypeByPredicateInChain finds an EntityType in the entity type and its parent chain
// based on the provided predicate function.
// If no EntityType matches the predicate, it returns nil.
func (e *EntityType) FindEntityTypeByPredicateInChain(predicate func(*EntityType) bool) *EntityType {
	root := e
	for root != nil {
		if predicate(root) {
			return root
		}
		root = root.parent
	}
	return nil
}

// FindTraitsSchemaInChain finds the traits schema in the entity type and its parent chain.
// If no traits schema is found, it returns nil.
func (e *EntityType) FindTraitsSchemaInChain() *jsonschema.JSONSchemaCTI {
	root := e
	for root != nil {
		if root.TraitsSchema != nil {
			return root.TraitsSchema
		}
		root = root.parent
	}
	return nil
}

// GetTraits returns the traits of the entity type.
func (e *EntityType) GetTraits() map[string]any {
	return e.Traits
}

// GetRawSchema returns the raw JSON schema of the entity type.
// It marshals the Schema field to JSON bytes and caches the result.
func (e *EntityType) GetRawSchema() ([]byte, error) {
	if e.rawSchema == nil {
		if b, err := json.Marshal(e.Schema); err == nil {
			e.rawSchema = b
		} else {
			return nil, fmt.Errorf("marshal values: %w", err)
		}
	}
	return e.rawSchema, nil
}

// GetRawTraits returns the raw JSON traits of the entity type.
// It marshals the Traits field to JSON bytes and caches the result.
func (e *EntityType) GetRawTraits() ([]byte, error) {
	if e.rawTraitValues == nil {
		if b, err := json.Marshal(e.Traits); err == nil {
			e.rawTraitValues = b
		} else {
			return nil, fmt.Errorf("marshal values: %w", err)
		}
	}
	return e.rawTraitValues, nil
}

// GetMergedTraits returns the merged traits of the entity type and its parent chain.
// It combines traits from the entity type and its parent entities, ensuring that traits from parent entities
// are included only if they are not already defined in the current entity type.
//
// Method provides lazy initialization of the merged traits and caches the result.
// Use ResetMergedTraits to reset the cached traits if necessary.
func (e *EntityType) GetMergedTraits() map[string]any {
	if e.mergedTraits != nil {
		return e.mergedTraits
	}
	mergedTraits := make(map[string]any)
	root := e
	for root != nil {
		if root.Traits != nil {
			for k, v := range root.Traits {
				if _, exists := mergedTraits[k]; !exists {
					mergedTraits[k] = v
				}
			}
		}
		root = root.parent
	}

	e.mergedTraits = mergedTraits

	return mergedTraits
}

// ResetMergedTraits resets the cached merged traits of the entity type.
// This is useful when Traits have been modified and need to be reloaded.
func (e *EntityType) ResetMergedTraits() {
	e.mergedTraits = nil
}

// ValidateBytes validates the values against the entity type schema.
func (e *EntityType) ValidateBytes(j []byte) error {
	s, err := e.GetSchemaValidator()
	if err != nil {
		return fmt.Errorf("get schema validator: %w", err)
	}
	return jsonschema.ValidateWrapper(s, gojsonschema.NewBytesLoader(j))
}

// Validate validates the values against the entity type schema.
//
// Note that this function will first marshal Go interface into bytes and then
// unmarshal again. If your data already comes in bytes, consider using ValidateBytes
// for better performance.
func (e *EntityType) Validate(j any) error {
	s, err := e.GetSchemaValidator()
	if err != nil {
		return fmt.Errorf("get schema validator: %w", err)
	}
	return jsonschema.ValidateWrapper(s, gojsonschema.NewGoLoader(j))
}

// GetSchemaValidator compiles, caches and returns the validator schema based on the merged schema.
//
// This method is useful only for metadata validation. For payloads validation, please use Validate and ValidateBytes
// methods.
func (e *EntityType) GetSchemaValidator() (*gojsonschema.Schema, error) {
	if e.validatorSchema != nil {
		return e.validatorSchema, nil
	}
	mergedSchema, err := e.GetMergedSchema()
	if err != nil {
		return nil, fmt.Errorf("get merged schema: %w", err)
	}
	s, err := jsonschema.CompileJSONSchemaCTIWithValidation(mergedSchema)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	e.validatorSchema = s
	return e.validatorSchema, nil
}

// GetTraitsSchemaValidator compiles, caches and returns the traits schema validator.
//
// This method is useful only for metadata validation.
func (e *EntityType) GetTraitsSchemaValidator() (*gojsonschema.Schema, error) {
	if e.validatorTraitsSchema != nil {
		return e.validatorTraitsSchema, nil
	}
	s, err := jsonschema.CompileJSONSchemaCTIWithValidation(e.TraitsSchema)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	e.validatorTraitsSchema = s
	return e.validatorTraitsSchema, nil
}

// ReplacePointer replaces the pointer receiver with the provided object.
func (e *EntityType) ReplacePointer(src Entity) error {
	switch src := src.(type) {
	case *EntityType:
		*e = *src
	default:
		return errors.New("invalid type for EntityType replacement")
	}
	return nil
}

// SetSchema sets the schema for the entity type.
func (e *EntityType) SetSchema(schema *jsonschema.JSONSchemaCTI) {
	e.Schema = schema
}

// SetTraitsSchema sets the traits schema and annotations for the entity type.
func (e *EntityType) SetTraitsSchema(traitsSchema *jsonschema.JSONSchemaCTI, traitsAnnotations map[GJsonPath]*Annotations) {
	e.TraitsSchema = traitsSchema
	e.TraitsAnnotations = traitsAnnotations
}

// SetTraits sets the traits for the entity type.
func (e *EntityType) SetTraits(traits map[string]any) {
	e.Traits = traits
}

// SetSourceMap sets the source map for the entity type.
func (e *EntityType) SetSourceMap(sourceMap *TypeSourceMap) {
	e.SourceMap = sourceMap
}

// SetTraitsSourceMap sets the source map for the traits schema.
func (e *EntityType) SetTraitsSourceMap(sourceMap *TypeSourceMap) {
	e.TraitsSourceMap = sourceMap
}

// IsNil checks if the underlying pointer receiver is nil.
func (e *EntityType) IsNil() bool {
	return e == nil
}

// NewEntityInstance creates a new EntityInstance with the given CTI identifier and values.
func NewEntityInstance(id string, values any) (*EntityInstance, error) {
	if id == "" {
		return nil, errors.New("identifier is empty")
	}
	if values == nil {
		return nil, errors.New("values is nil")
	}

	obj := &EntityInstance{
		entity: entity{
			CTI:         id,
			Final:       true, // All entities are final by default
			Access:      consts.AccessModifierProtected,
			Annotations: make(map[GJsonPath]*Annotations),
		},
		Values: values,
	}

	return obj, nil
}

// EntityInstance represents a CTI entity instance with values that conform to a specific EntityType.
// It is used to represent a specific instance of a domain type with its own data.
// For more information, see [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#instances
type EntityInstance struct {
	entity `yaml:",inline"`

	// Values is the values of the entity instance.
	// Can be any value that conforms to the schema of the parent EntityType.
	Values any `json:"values" yaml:"values"`

	// rawValues is the cached marshaled value of Values.
	rawValues []byte `json:"-" yaml:"-"`

	// SourceMap is the information about the source of the entity instance.
	SourceMap *InstanceSourceMap `json:"source_map,omitempty" yaml:"source_map,omitempty"`
}

type InstanceSourceMap struct {
	// AnnotationType is the information about the annotation type that was used to define the instance.
	AnnotationType AnnotationType `json:"$annotationType,omitempty" yaml:"$annotationType,omitempty"`

	DocumentSourceMap `yaml:",inline"`
}

// SetParent sets the parent EntityType for the entity instance.
// It allows setting a parent to final EntityType since instances are allowed to have final parent type.
// If provided parent entity is nil, it removes the parent reference.
func (e *EntityInstance) SetParent(entity *EntityType) error {
	if entity == nil {
		e.parent = nil
		return nil
	}
	e.parent = entity
	return nil
}

// FindEntityTypeByPredicateInChain finds an EntityType in the entity type and its parent chain
// based on the provided predicate function.
// If no EntityType matches the predicate, it returns nil.
func (e *EntityInstance) FindEntityTypeByPredicateInChain(predicate func(*EntityType) bool) *EntityType {
	root := e.Parent()
	for root != nil {
		if predicate(root) {
			return root
		}
		root = root.parent
	}
	return nil
}

// GetRawValues returns the raw JSON values of the entity instance.
// It marshals the Values field to JSON bytes and caches the result.
func (e *EntityInstance) GetRawValues() ([]byte, error) {
	if e.rawValues == nil {
		if b, err := json.Marshal(e.Values); err == nil {
			e.rawValues = b
		} else {
			return nil, fmt.Errorf("marshal values: %w", err)
		}
	}
	return e.rawValues, nil
}

// GetValueByAttributeSelector returns the value of the entity instance by the given attribute selector.
// It uses the attribute selector to navigate through the Values map and retrieve the value.
// To be able to use this method, the Values field must be a map[string]any.
// For more information, see [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#attribute-selector
func (e *EntityInstance) GetValueByAttributeSelector(attributeSelector string) (any, error) {
	as, err := attribute_selector.NewAttributeSelector(attributeSelector)
	if err != nil {
		return nil, fmt.Errorf("create attribute selector: %w", err)
	}
	v, ok := e.Values.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("values are not a map[string]any: %T", e.Values)
	}
	return as.WalkJSON(v)
}

// ReplacePointer replaces the pointer receiver with the provided object.
func (e *EntityInstance) ReplacePointer(src Entity) error {
	switch src := src.(type) {
	case *EntityInstance:
		*e = *src
	default:
		return errors.New("invalid type for EntityInstance replacement")
	}
	return nil
}

// SetSourceMap sets the source map for the entity instance.
func (e *EntityInstance) SetSourceMap(sourceMap *InstanceSourceMap) {
	e.SourceMap = sourceMap
}

// IsNil checks if the underlying pointer receiver is nil.
func (e *EntityInstance) IsNil() bool {
	return e == nil
}
