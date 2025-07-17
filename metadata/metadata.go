package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata/attribute_selector"
	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/tidwall/gjson"
)

type Entities []Entity
type EntityTypeMap map[string]*EntityType
type EntityInstanceMap map[string]*EntityInstance
type EntityMap map[string]Entity

// TODO: For future use. Need to create a context and move all creation methods there.
type MContext struct{}

// PtrReplacer is an interface for objects that can replace their pointers with another object of the same type.
type PtrReplacer[T any] interface {
	ReplacePointer(T) error
}

type NilChecker interface {
	IsNil() bool
}

type Entity interface {
	GetCTI() string

	SetFinal(final bool)
	IsFinal() bool

	GetAccess() consts.AccessModifier
	SetAccess(access consts.AccessModifier)
	IsSamePackage(other Entity) bool
	IsSameVendor(other Entity) bool
	IsAccessibleBy(other Entity) error

	IsA(other *EntityType) bool

	SetResilient(resilient bool)
	SetDisplayName(displayName string)
	SetDescription(description string)
	SetDictionaries(dictionaries map[string]any)

	Parent() *EntityType
	SetParent(*EntityType) error

	Expression() (*cti.Expression, error)
	Match(other Entity) (bool, error)
	Vendor() string
	Package() string
	Name() string
	Version() cti.Version

	Context() *MContext

	GetAnnotations() map[GJsonPath]*Annotations
	FindAnnotationsByPredicateInChain(key GJsonPath, predicate func(*Annotations) bool) *Annotations
	FindAnnotationsByKeyInChain(key GJsonPath) *Annotations

	NilChecker
}

type Annotations struct {
	CTI           any                   `json:"cti.cti,omitempty" yaml:"cti.cti,omitempty"` // string or []string
	ID            *bool                 `json:"cti.id,omitempty" yaml:"cti.id,omitempty"`   // bool?
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

func (a Annotations) ReadCTI() []string {
	if a.CTI == nil {
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

func (a Annotations) ReadCTISchema() []string {
	if a.Schema == nil {
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

func (a Annotations) ReadReference() []string {
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

// Base properties for all CTI entities.
type entity struct {
	// TODO: Add UUID (computable)
	// TODO: Add IsAnonymous method
	// TODO: Implement Validate method

	Final        bool                       `json:"final" yaml:"final"`
	Access       consts.AccessModifier      `json:"access" yaml:"access"`
	CTI          string                     `json:"cti" yaml:"cti"`
	Resilient    bool                       `json:"resilient" yaml:"resilient"`
	DisplayName  string                     `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description  string                     `json:"description,omitempty" yaml:"description,omitempty"`
	Dictionaries map[string]any             `json:"dictionaries,omitempty" yaml:"dictionaries,omitempty"`
	Annotations  map[GJsonPath]*Annotations `json:"annotations,omitempty" yaml:"annotations,omitempty"`

	parent *EntityType `json:"-" yaml:"-"` // Parent entity type, if any

	expression *cti.Expression `json:"-" yaml:"-"` // Parsed CTI expression, if any

	ctx *MContext `json:"-" yaml:"-"` // For future reflection purposes
}

type EntitySourceMap struct {
	// SourcePath is a relative path to the RAML file where the CTI parent is defined.
	SourcePath string `json:"$sourcePath,omitempty" yaml:"$sourcePath,omitempty"`

	// OriginalPath is a relative path to RAML fragment where the CTI entity is defined.
	OriginalPath string `json:"$originalPath,omitempty" yaml:"$originalPath,omitempty"`
}

func (e *entity) GetCTI() string {
	return e.CTI
}

func (e *entity) GetAccess() consts.AccessModifier {
	return e.Access
}

func (e *entity) Parent() *EntityType {
	return e.parent
}

func (e *entity) GetAnnotations() map[GJsonPath]*Annotations {
	return e.Annotations
}

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

func (e *entity) Context() *MContext {
	return e.ctx
}

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

func (e *entity) IsSameVendor(other Entity) bool {
	if other == nil {
		return false
	}
	if e.Vendor() != other.Vendor() {
		return false
	}
	return true
}

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

func (e *entity) Vendor() string {
	expr, err := e.Expression()
	if err != nil {
		return ""
	}
	tail := expr.Tail()
	return string(tail.Vendor)
}

func (e *entity) Package() string {
	expr, err := e.Expression()
	if err != nil {
		return ""
	}
	tail := expr.Tail()
	return string(tail.Package)
}

func (e *entity) Name() string {
	expr, err := e.Expression()
	if err != nil {
		return ""
	}
	tail := expr.Tail()
	return string(tail.EntityName)
}

func (e *entity) Version() cti.Version {
	expr, err := e.Expression()
	if err != nil {
		return cti.Version{}
	}
	tail := expr.Tail()
	return tail.Version
}

func (e *entity) SetParent(_ *EntityType) error {
	return errors.New("entity does not implement SetParent")
}

func (e *entity) ReplacePointer(_ Entity) error {
	return errors.New("entity does not implement ReplacePointer")
}

func (e *entity) IsFinal() bool {
	return e.Final
}

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

func (e *entity) IsA(entity *EntityType) bool {
	if entity == nil {
		return false
	}
	return strings.HasPrefix(e.CTI, entity.CTI)
}

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

func (e *entity) SetFinal(final bool) {
	e.Final = final
}

func (e *entity) SetAccess(access consts.AccessModifier) {
	e.Access = access
}

func (e *entity) SetResilient(resilient bool) {
	e.Resilient = resilient
}

func (e *entity) SetDisplayName(displayName string) {
	e.DisplayName = displayName
}

func (e *entity) SetDescription(description string) {
	e.Description = description
}

func (e *entity) SetDictionaries(dictionaries map[string]any) {
	e.Dictionaries = dictionaries
}

func (e *entity) SetAnnotations(annotations map[GJsonPath]*Annotations) {
	e.Annotations = annotations
}

func (e *entity) IsNil() bool {
	return e == nil
}

func NewEntityType(
	id string,
	schema *jsonschema.JSONSchemaCTI,
	annotations map[GJsonPath]*Annotations,
) (*EntityType, error) {
	switch {
	case schema == nil:
		return nil, errors.New("schema is nil")
	case annotations == nil:
		return nil, errors.New("annotations are nil")
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

type EntityType struct {
	entity `yaml:",inline"`

	Schema            *jsonschema.JSONSchemaCTI  `json:"schema" yaml:"schema"`
	TraitsSchema      *jsonschema.JSONSchemaCTI  `json:"traits_schema,omitempty" yaml:"traits_schema,omitempty"`
	TraitsAnnotations map[GJsonPath]*Annotations `json:"traits_annotations,omitempty" yaml:"traits_annotations,omitempty"`
	Traits            map[string]any             `json:"traits,omitempty" yaml:"traits,omitempty"`

	rawSchema []byte `json:"-" yaml:"-"` // Cached raw schema
	// FIXME: Need to remove. Raw values are only needed for GJSON, but it works with bytes.
	// Need custom visitor for Go interface based on GJSON.
	rawTraitValues []byte `json:"-" yaml:"-"` // Cached raw trait values

	SourceMap EntityTypeSourceMap `json:"source_map,omitempty" yaml:"source_map,omitempty"`
}

type EntityTypeSourceMap struct {
	Name            string `json:"$name,omitempty" yaml:"$name,omitempty"`
	EntitySourceMap `yaml:",inline"`
}

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

func (e *EntityType) GetMergedSchema() (*jsonschema.JSONSchemaCTI, error) {
	if e.Schema == nil {
		return nil, errors.New("entity type schema is nil")
	}
	if e.parent == nil {
		return e.Schema, nil
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

	return &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Version:     "http://json-schema.org/draft-07/schema",
			Ref:         origSelfRefType,
			Definitions: definitions,
		},
	}, nil
}

func (e *EntityType) GetTraitsSchema() *jsonschema.JSONSchemaCTI {
	return e.TraitsSchema
}

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

func (e *EntityType) FindEntityTypeByPredicate(predicate func(*EntityType) bool) *EntityType {
	root := e
	for root != nil {
		if predicate(root) {
			return root
		}
		root = root.parent
	}
	return nil
}

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

func (e *EntityType) GetTraits() map[string]any {
	return e.Traits
}

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

func (e *EntityType) FindTraitsInChain() map[string]any {
	root := e
	for root != nil {
		if root.Traits != nil {
			return root.Traits
		}
		root = root.parent
	}
	return nil
}

func (e *EntityType) Validate() error {
	// TODO: Implement
	return nil
}

func (e *EntityType) ReplacePointer(src Entity) error {
	switch src := src.(type) {
	case *EntityType:
		*e = *src
	default:
		return errors.New("invalid type for EntityType replacement")
	}
	return nil
}

func (e *EntityType) SetSchema(schema *jsonschema.JSONSchemaCTI) {
	e.Schema = schema
}

func (e *EntityType) SetTraitsSchema(traitsSchema *jsonschema.JSONSchemaCTI, traitsAnnotations map[GJsonPath]*Annotations) {
	e.TraitsSchema = traitsSchema
	e.TraitsAnnotations = traitsAnnotations
}

func (e *EntityType) SetTraits(traits map[string]any) {
	e.Traits = traits
}

func (e *EntityType) SetSourceMap(sourceMap EntityTypeSourceMap) {
	e.SourceMap = sourceMap
}

func (e *EntityType) IsNil() bool {
	return e == nil
}

func NewEntityInstance(id string, values any) (*EntityInstance, error) {
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

type EntityInstance struct {
	entity `yaml:",inline"`

	Values any `json:"values" yaml:"values"`

	// FIXME: Need to remove. Raw values are only needed for GJSON, but it works with bytes.
	// Need custom visitor for Go interface based on GJSON.
	rawValues []byte                  `json:"-" yaml:"-"`
	SourceMap EntityInstanceSourceMap `json:"source_map,omitempty" yaml:"source_map,omitempty"`
}

type EntityInstanceSourceMap struct {
	AnnotationType  AnnotationType `json:"$annotationType,omitempty" yaml:"$annotationType,omitempty"`
	EntitySourceMap `yaml:",inline"`
}

func (e *EntityInstance) SetParent(entity *EntityType) error {
	if entity == nil {
		e.parent = nil
		return nil
	}
	e.parent = entity
	return nil
}

func (e *EntityInstance) FindEntityTypeByPredicate(predicate func(*EntityType) bool) *EntityType {
	root := e.Parent()
	for root != nil {
		if predicate(root) {
			return root
		}
		root = root.parent
	}
	return nil
}

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

func (e *EntityInstance) Validate() error {
	return nil
}

func (e *EntityInstance) ValidateValues() error {
	return nil
}

func (e *EntityInstance) ReplacePointer(src Entity) error {
	switch src := src.(type) {
	case *EntityInstance:
		*e = *src
	default:
		return errors.New("invalid type for EntityInstance replacement")
	}
	return nil
}

func (e *EntityInstance) SetSourceMap(sourceMap EntityInstanceSourceMap) {
	e.SourceMap = sourceMap
}

func (e *EntityInstance) IsNil() bool {
	return e == nil
}
