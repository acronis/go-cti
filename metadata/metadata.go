package metadata

import (
	"errors"
	"fmt"

	"github.com/acronis/go-cti"
	"github.com/tidwall/gjson"
)

type Entities []Entity
type EntityTypeMap map[string]*EntityType
type EntityInstanceMap map[string]*EntityInstance
type EntityMap map[string]Entity

type MContext struct{}

type Version struct {
	Major uint
	Minor uint
}

// PtrReplacer is an interface for objects that can replace their pointers with another object of the same type.
type PtrReplacer[T any] interface {
	ReplacePointer(T) error
}

type Entity interface {
	GetCti() string
	IsFinal() bool

	Parent() *EntityType
	SetParent(*EntityType) error

	Children() Entities
	GetChild(cti string) Entity
	AddChild(Entity) error

	Version() Version
	// TODO: Maybe it would make sense to move the object version into corresponding types
	GetObjectMajorVersion(major uint) Entity
	GetObjectVersion(major uint, minor uint) Entity
	GetObjectVersions() map[Version]Entity
	AddObjectVersion(Entity) error

	Expression() *cti.Expression
	Context() *MContext

	GetAnnotations() map[GJsonPath]Annotations
	FindAnnotationsInChain(predicate func(*Annotations) bool) *Annotations
	FindAnnotationsKeyInChain(key GJsonPath) *Annotations
}

type Annotations struct {
	Cti           interface{}            `json:"cti.cti,omitempty"` // string or []string
	ID            *bool                  `json:"cti.id,omitempty"`  // bool?
	DisplayName   *bool                  `json:"cti.display_name,omitempty"`
	Description   *bool                  `json:"cti.description,omitempty"`
	Reference     interface{}            `json:"cti.reference,omitempty"` // bool or string or []string
	Overridable   *bool                  `json:"cti.overridable,omitempty"`
	Final         *bool                  `json:"cti.final,omitempty"`
	Resilient     *bool                  `json:"cti.resilient,omitempty"`
	Asset         *bool                  `json:"cti.asset,omitempty"`
	L10N          *bool                  `json:"cti.l10n,omitempty"`
	Schema        interface{}            `json:"cti.schema,omitempty"` // string or []string
	Meta          string                 `json:"cti.meta,omitempty"`
	PropertyNames map[string]interface{} `json:"cti.propertyNames,omitempty"`
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

type EntityOption func(*entity) error

func WithFinal(final bool) EntityOption {
	return func(obj *entity) error {
		obj.Final = final
		return nil
	}
}

func WithResilient(resilient bool) EntityOption {
	return func(obj *entity) error {
		obj.Resilient = resilient
		return nil
	}
}

func WithDisplayName(displayName string) EntityOption {
	return func(obj *entity) error {
		obj.DisplayName = displayName
		return nil
	}
}

func WithDescription(description string) EntityOption {
	return func(obj *entity) error {
		obj.Description = description
		return nil
	}
}

func WithDictionaries(dictionaries map[string]interface{}) EntityOption {
	return func(obj *entity) error {
		obj.Dictionaries = dictionaries
		return nil
	}
}

// Base properties for all CTI entities.
type entity struct {
	// TODO: Add UUID (computable)
	// TODO: Add IsAnonymous method
	// TODO: Add Validate and GetMergedSchema methods with merged schema caching

	Final        bool                      `json:"final"`
	Cti          string                    `json:"cti"`
	Resilient    bool                      `json:"resilient"`
	DisplayName  string                    `json:"display_name,omitempty"`
	Description  string                    `json:"description,omitempty"`
	Dictionaries map[string]interface{}    `json:"dictionaries,omitempty"`
	Annotations  map[GJsonPath]Annotations `json:"annotations"`

	// Form a doubly-linked list of Entities
	parent   *EntityType `json:"-"`
	children Entities    `json:"-"`

	// CTI Parser properties
	// TODO: Probably move out to separate implementation
	vendor     string          `json:"-"`
	pkg        string          `json:"-"`
	name       string          `json:"-"`
	version    Version         `json:"-"`
	expression *cti.Expression `json:"-"`

	// Adjacent list of versions that are relevant to this object.
	versions map[Version]Entity `json:"-"`

	// TODO: Add extensible SourceMap interface

	ctx *MContext `json:"-"` // For future reflection purposes
}

type EntitySourceMap struct {
	// SourcePath is a relative path to the RAML file where the CTI parent is defined.
	SourcePath string `json:"$sourcePath,omitempty"`

	// OriginalPath is a relative path to RAML fragment where the CTI entity is defined.
	OriginalPath string `json:"$originalPath,omitempty"`
}

func (e *entity) GetCti() string {
	return e.Cti
}

func (e *entity) Parent() *EntityType {
	return e.parent
}

func (e *entity) Children() Entities {
	return e.children
}

func (e *entity) GetObjectVersions() map[Version]Entity {
	return e.versions
}

func (e *entity) GetAnnotations() map[GJsonPath]Annotations {
	return e.Annotations
}

func (e *entity) FindAnnotationsInChain(predicate func(*Annotations) bool) *Annotations {
	var root Entity = e
	for root != nil {
		annotations := root.GetAnnotations()
		for _, val := range annotations {
			if predicate(&val) {
				return &val
			}
		}
		root = e.parent
	}
	return nil
}

func (e *entity) FindAnnotationsKeyInChain(key GJsonPath) *Annotations {
	var root Entity = e
	for root != nil {
		annotations := root.GetAnnotations()
		if val, ok := annotations[key]; ok {
			return &val
		}
		root = e.parent
	}
	return nil
}

func (e *entity) Context() *MContext {
	return e.ctx
}

func (e *entity) GetChild(cti string) Entity {
	// TODO: Implement FindChild
	for _, child := range e.children {
		if child.GetCti() == cti {
			return child
		}
	}
	return nil
}

func (e *entity) Version() Version {
	return e.version
}

func (e *entity) GetObjectMajorVersion(_ uint) Entity {
	// TODO: Implement
	return nil
}

func (e *entity) GetObjectVersion(major uint, minor uint) Entity {
	return e.versions[Version{Major: major, Minor: minor}]
}

func (e *entity) AddObjectVersion(object Entity) error {
	// TODO: Implement more sophisticated checks
	if object == nil {
		return errors.New("object is nil")
	}
	expr := object.Expression()
	if expr == nil {
		return errors.New("entity expression is nil")
	}
	ver := object.Version()
	if ver.Major == 0 && ver.Minor == 0 {
		return errors.New("object version is not set")
	}
	if _, ok := e.versions[ver]; ok {
		return errors.New("object with the same version already exists")
	}
	e.versions[ver] = object
	return nil
}

func (e *entity) SetParent(_ *EntityType) error {
	return errors.New("entity does not implement SetParent")
}

func (e *entity) AddChild(_ Entity) error {
	return errors.New("entity does not implement AddChild")
}

func (e *entity) ReplacePointer(_ Entity) error {
	return errors.New("entity does not implement ReplacePointer")
}

func (e *entity) IsFinal() bool {
	return e.Final
}

func (e *entity) Expression() *cti.Expression {
	return e.expression
}

type EntityTypeOption func(*EntityType) error

func WithTraitsSchema(schema map[string]interface{}, annotations map[GJsonPath]Annotations) EntityTypeOption {
	return func(obj *EntityType) error {
		obj.TraitsSchema = schema
		obj.TraitsAnnotations = annotations
		return nil
	}
}

func WithTraits(traits interface{}) EntityTypeOption {
	return func(obj *EntityType) error {
		obj.Traits = traits
		return nil
	}
}

func WithTypeSourceMap(sourceMap *EntityTypeSourceMap) EntityTypeOption {
	return func(obj *EntityType) error {
		if sourceMap == nil {
			return errors.New("source map is nil")
		}
		obj.SourceMap = *sourceMap
		return nil
	}
}

func NewEntityType(
	id string,
	schema map[string]interface{},
	annotations map[GJsonPath]Annotations,
	commonOptions []EntityOption,
	specificOptions []EntityTypeOption,
) (*EntityType, error) {
	expr, err := cti.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse cti: %w", err)
	}
	return NewEntityTypeFromExpr(&expr, schema, annotations, commonOptions, specificOptions)
}

func NewEntityTypeFromExpr(
	expr *cti.Expression,
	schema map[string]interface{},
	annotations map[GJsonPath]Annotations,
	commonOptions []EntityOption,
	specificOptions []EntityTypeOption,
) (*EntityType, error) {
	switch {
	case expr == nil:
		return nil, errors.New("expression is nil")
	case schema == nil:
		return nil, errors.New("schema is nil")
	case annotations == nil:
		return nil, errors.New("annotations are nil")
	}

	tail := expr.Tail()
	obj := &EntityType{
		entity: entity{
			Cti:         expr.String(), // TODO: This potentially introduces unwanted overhead since we are reconstructing already known string
			Final:       true,          // All entities are final by default
			Annotations: annotations,

			vendor: string(tail.Vendor),
			pkg:    string(tail.Package),
			name:   string(tail.EntityName),
			version: Version{
				Major: tail.Version.Major.Value,
				Minor: tail.Version.Minor.Value,
			},
			expression: expr,

			versions: make(map[Version]Entity),
		},
		Schema: schema,
	}

	for _, option := range commonOptions {
		if err := option(&obj.entity); err != nil {
			return nil, fmt.Errorf("common option failed: %w", err)
		}
	}

	for _, option := range specificOptions {
		if err := option(obj); err != nil {
			return nil, fmt.Errorf("specific option failed: %w", err)
		}
	}

	return obj, nil
}

type EntityType struct {
	entity

	Schema            map[string]interface{}    `json:"schema"`
	TraitsSchema      map[string]interface{}    `json:"traits_schema,omitempty"`
	TraitsAnnotations map[GJsonPath]Annotations `json:"traits_annotations,omitempty"`
	Traits            interface{}               `json:"traits,omitempty"`

	SourceMap EntityTypeSourceMap `json:"source_map,omitempty"`
}

type EntityTypeSourceMap struct {
	Name string `json:"$name,omitempty"`
	EntitySourceMap
}

func (e *EntityType) SetParent(entity *EntityType) error {
	// TODO: Implement more sophisticated checks
	if entity == nil {
		e.parent = nil
		return nil
	}
	ver := entity.Version()
	if ver.Major == 0 && ver.Minor == 0 {
		return errors.New("type version is not set")
	}
	if e.expression == nil {
		return errors.New("entity expression is nil")
	}
	ok, err := entity.expression.MatchIgnoreQuery(*e.expression)
	if err != nil {
		return fmt.Errorf("failed to match expression: %w", err)
	} else if !ok {
		return fmt.Errorf("expression %s does not match %s", e.expression, entity.expression)
	}
	if entity.IsFinal() {
		return errors.New("cannot set parent to a final type")
	}
	e.parent = entity
	return nil
}

func (e *EntityType) MergeSchemaChain() map[string]interface{} {
	return nil
}

func (e *EntityType) GetTraitsSchema() interface{} {
	return e.TraitsSchema
}

func (e *EntityType) FindTraitsSchemaInChain() map[string]interface{} {
	root := e
	for root != nil {
		if root.TraitsSchema != nil {
			return root.TraitsSchema
		}
		root = e.parent
	}
	return nil
}

func (e *EntityType) GetTraits() interface{} {
	return e.Traits
}

func (e *EntityType) FindTraitsInChain() interface{} {
	root := e
	for root != nil {
		if root.Traits != nil {
			return root.Traits
		}
		root = e.parent
	}
	return nil
}

func (e *EntityType) Validate() error {
	return nil
}

func (e *EntityType) AddChild(object Entity) error {
	e.children = append(e.children, object)
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

// For EntityInstance
type EntityInstanceOption func(*EntityInstance) error

func WithInstanceSourceMap(sourceMap *EntityInstanceSourceMap) EntityInstanceOption {
	return func(obj *EntityInstance) error {
		if sourceMap == nil {
			return errors.New("source map is nil")
		}
		obj.SourceMap = *sourceMap
		return nil
	}
}

func NewEntityInstance(
	id string,
	values interface{},
	commonOptions []EntityOption,
	specificOptions []EntityInstanceOption,
) (*EntityInstance, error) {
	expr, err := cti.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse cti: %w", err)
	}
	return NewEntityInstanceFromExpr(&expr, values, commonOptions, specificOptions)
}

func NewEntityInstanceFromExpr(
	expr *cti.Expression,
	values interface{},
	commonOptions []EntityOption,
	specificOptions []EntityInstanceOption,
) (*EntityInstance, error) {
	switch {
	case expr == nil:
		return nil, errors.New("expression is nil")
	case values == nil:
		return nil, errors.New("values is nil")
	}

	tail := expr.Tail()
	obj := &EntityInstance{
		entity: entity{
			Cti:         expr.String(), // TODO: This potentially introduces unwanted overhead since we are reconstructing already known string
			Final:       true,          // All instances are final by default
			Annotations: make(map[GJsonPath]Annotations),

			vendor: string(tail.Vendor),
			pkg:    string(tail.Package),
			name:   string(tail.EntityName),
			version: Version{
				Major: tail.Version.Major.Value,
				Minor: tail.Version.Minor.Value,
			},
			expression: expr,

			versions: make(map[Version]Entity),
		},
		Values: values,
	}

	for _, option := range commonOptions {
		if err := option(&obj.entity); err != nil {
			return nil, fmt.Errorf("common option failed: %w", err)
		}
	}

	for _, option := range specificOptions {
		if err := option(obj); err != nil {
			return nil, fmt.Errorf("specific option failed: %w", err)
		}
	}

	return obj, nil
}

type EntityInstance struct {
	entity

	Values interface{} `json:"values"`

	SourceMap EntityInstanceSourceMap `json:"source_map,omitempty"`
}

type EntityInstanceSourceMap struct {
	AnnotationType AnnotationType `json:"$annotationType,omitempty"`
	EntitySourceMap
}

func (e *EntityInstance) SetParent(entity *EntityType) error {
	// TODO: Implement more sophisticated checks
	if entity == nil {
		e.parent = nil
		return nil
	}
	ver := entity.Version()
	if ver.Major == 0 && ver.Minor == 0 {
		return errors.New("type version is not set")
	}
	ok, err := entity.expression.MatchIgnoreQuery(*e.expression)
	if err != nil {
		return fmt.Errorf("failed to match expression: %w", err)
	} else if !ok {
		return fmt.Errorf("expression %s does not match %s", e.expression, entity.expression)
	}
	e.parent = entity
	return nil
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

func (e *EntityInstance) AddChild(_ Entity) error {
	return errors.New("EntityInstance does not support children")
}
