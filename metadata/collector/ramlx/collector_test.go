package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/filesys"
	"github.com/acronis/go-cti/metadata/ramlx"
	"github.com/acronis/go-raml/v2"
	"github.com/stretchr/testify/require"
)

// entityRAML prepends the standard RAML 1.0 Library header with cti annotation imports
// to the given body, producing a complete entity RAML file content.
func entityRAML(body string) string {
	return "#%RAML 1.0 Library\n\nuses:\n  cti: .ramlx/cti.raml\n\n" + body
}

// setupRAMLxSpec extracts the embedded .ramlx specification files into <dir>/.ramlx.
func setupRAMLxSpec(t *testing.T, dir string) {
	t.Helper()
	dst := filepath.Join(dir, ".ramlx")
	require.NoError(t, os.MkdirAll(dst, 0755))
	require.NoError(t, filesys.CopyFS(ramlx.RamlFiles, dst, filesys.WithRoot("spec_v1")))
}

// newTestCollector writes entityFiles into a temp directory, installs the .ramlx spec,
// generates a synthetic index RAML library that uses those files, parses it,
// and returns the resulting RAMLXCollector ready for Collect().
//
// Keys of entityFiles are file names relative to the temp directory (e.g. "entities.raml").
// Values are complete RAML 1.0 Library file contents (use entityRAML() for the standard header).
func newTestCollector(t *testing.T, entityFiles map[string]string) *RAMLXCollector {
	t.Helper()

	tmpDir := t.TempDir()
	setupRAMLxSpec(t, tmpDir)

	// Sort names for deterministic index.raml generation across test runs.
	names := make([]string, 0, len(entityFiles))
	for name := range entityFiles {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	sb.WriteString("#%RAML 1.0 Library\nuses:")
	for i, name := range names {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(entityFiles[name]), 0644))
		sb.WriteString(fmt.Sprintf("\n  e%d: %s", i+1, name))
	}

	r, err := raml.ParseFromString(sb.String(), "index.raml", tmpDir, raml.OptWithValidate())
	require.NoError(t, err)

	c, err := NewRAMLXCollector(r)
	require.NoError(t, err)
	return c
}

// --- Constructor tests ---

func TestNewRAMLXCollector_NilRAML(t *testing.T) {
	_, err := NewRAMLXCollector(nil)
	require.ErrorContains(t, err, "raml is nil")
}

func TestNewRAMLXCollector_NonLibraryEntryPoint(t *testing.T) {
	// DataType fragment is not a Library — the collector must reject it.
	r, err := raml.ParseFromString("#%RAML 1.0 DataType\ntype: string", "mytype.raml", t.TempDir())
	require.NoError(t, err)
	_, err = NewRAMLXCollector(r)
	require.ErrorContains(t, err, "entry point is not a library")
}

func TestNewRAMLXCollector_ValidLibrary(t *testing.T) {
	r, err := raml.ParseFromString("#%RAML 1.0 Library\nuses:", "index.raml", t.TempDir())
	require.NoError(t, err)
	c, err := NewRAMLXCollector(r)
	require.NoError(t, err)
	require.NotNil(t, c)
	require.NotNil(t, c.Registry)
}

// --- Basic collection tests ---

func TestCollect_EmptyLibrary(t *testing.T) {
	c := newTestCollector(t, nil)
	reg, err := c.Collect()
	require.NoError(t, err)
	require.Empty(t, reg.Types)
	require.Empty(t, reg.Instances)
	require.Empty(t, reg.Index)
}

func TestCollect_BasicTypes(t *testing.T) {
	tests := map[string]struct {
		raml      string
		ctiID     string
		wantTypes int
	}{
		"object type": {
			raml: entityRAML("types:\n  ObjectEntity:\n    (cti.cti): cti.x.y.object_entity.v1.0\n    type: object"),
			ctiID:     "cti.x.y.object_entity.v1.0",
			wantTypes: 1,
		},
		"string scalar type": {
			raml: entityRAML("types:\n  ScalarEntity:\n    (cti.cti): cti.x.y.scalar_entity.v1.0\n    type: string"),
			ctiID:     "cti.x.y.scalar_entity.v1.0",
			wantTypes: 1,
		},
		"nil type": {
			raml: entityRAML("types:\n  NilEntity:\n    (cti.cti): cti.x.y.nil_entity.v1.0\n    type: nil"),
			ctiID:     "cti.x.y.nil_entity.v1.0",
			wantTypes: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := newTestCollector(t, map[string]string{"entities.raml": tc.raml})
			reg, err := c.Collect()
			require.NoError(t, err)
			require.Len(t, reg.Types, tc.wantTypes)
			require.Contains(t, reg.Types, tc.ctiID)
			require.Contains(t, reg.Index, tc.ctiID)
		})
	}
}

func TestCollect_MultipleTypesInOneFile(t *testing.T) {
	ramlContent := entityRAML(`types:
  EntityA:
    (cti.cti): cti.x.y.entity_a.v1.0
    type: object
  EntityB:
    (cti.cti): cti.x.y.entity_b.v1.0
    type: string
  EntityC:
    (cti.cti): cti.x.y.entity_c.v1.0
    type: nil`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)
	require.Len(t, reg.Types, 3)
	require.Contains(t, reg.Types, "cti.x.y.entity_a.v1.0")
	require.Contains(t, reg.Types, "cti.x.y.entity_b.v1.0")
	require.Contains(t, reg.Types, "cti.x.y.entity_c.v1.0")
}

func TestCollect_TypesAcrossMultipleFiles(t *testing.T) {
	fileA := entityRAML("types:\n  EntityA:\n    (cti.cti): cti.x.y.entity_a.v1.0\n    type: object")
	fileB := entityRAML("types:\n  EntityB:\n    (cti.cti): cti.x.y.entity_b.v1.0\n    type: object")

	c := newTestCollector(t, map[string]string{
		"entities_a.raml": fileA,
		"entities_b.raml": fileB,
	})
	reg, err := c.Collect()
	require.NoError(t, err)
	require.Len(t, reg.Types, 2)
	require.Contains(t, reg.Types, "cti.x.y.entity_a.v1.0")
	require.Contains(t, reg.Types, "cti.x.y.entity_b.v1.0")
}

// --- Type annotation tests ---

func TestCollect_TypeAnnotations(t *testing.T) {
	tests := map[string]struct {
		raml            string
		ctiID           string
		wantFinal       bool
		wantAccess      consts.AccessModifier
		wantResilient   bool
		wantDisplayName string
		wantDescription string
	}{
		"final false explicit": {
			raml:      entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    (cti.final): false\n    type: object"),
			ctiID:     "cti.x.y.entity.v1.0",
			wantFinal:  false,
			wantAccess: consts.AccessModifierPublic,
		},
		"final true explicit": {
			raml:      entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    (cti.final): true\n    type: object"),
			ctiID:     "cti.x.y.entity.v1.0",
			wantFinal:  true,
			wantAccess: consts.AccessModifierPublic,
		},
		"final defaults to true": {
			// When (cti.final) is absent, NewEntityType sets Final=true by default.
			raml:      entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    type: object"),
			ctiID:     "cti.x.y.entity.v1.0",
			wantFinal:  true,
			wantAccess: consts.AccessModifierPublic,
		},
		"access public": {
			raml:      entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    (cti.access): public\n    type: object"),
			ctiID:     "cti.x.y.entity.v1.0",
			wantFinal:  true,
			wantAccess: consts.AccessModifierPublic,
		},
		"access protected": {
			raml:      entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    (cti.access): protected\n    type: object"),
			ctiID:     "cti.x.y.entity.v1.0",
			wantFinal:  true,
			wantAccess: consts.AccessModifierProtected,
		},
		"access private": {
			raml:      entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    (cti.access): private\n    type: object"),
			ctiID:     "cti.x.y.entity.v1.0",
			wantFinal:  true,
			wantAccess: consts.AccessModifierPrivate,
		},
		"resilient true": {
			raml:          entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    (cti.resilient): true\n    type: object"),
			ctiID:         "cti.x.y.entity.v1.0",
			wantFinal:     true,
			wantAccess:    consts.AccessModifierPublic,
			wantResilient: true,
		},
		"displayName from facet": {
			raml:            entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    displayName: My Custom Name\n    type: object"),
			ctiID:           "cti.x.y.entity.v1.0",
			wantFinal:       true,
			wantAccess:      consts.AccessModifierPublic,
			wantDisplayName: "My Custom Name",
		},
		"displayName defaults to type name": {
			raml:            entityRAML("types:\n  MyTypeName:\n    (cti.cti): cti.x.y.entity.v1.0\n    type: object"),
			ctiID:           "cti.x.y.entity.v1.0",
			wantFinal:       true,
			wantAccess:      consts.AccessModifierPublic,
			wantDisplayName: "MyTypeName",
		},
		"description": {
			raml:            entityRAML("types:\n  Entity:\n    (cti.cti): cti.x.y.entity.v1.0\n    description: My description text\n    type: object"),
			ctiID:           "cti.x.y.entity.v1.0",
			wantFinal:       true,
			wantAccess:      consts.AccessModifierPublic,
			wantDescription: "My description text",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := newTestCollector(t, map[string]string{"entities.raml": tc.raml})
			reg, err := c.Collect()
			require.NoError(t, err)

			entity := reg.Types[tc.ctiID]
			require.NotNil(t, entity)
			require.Equal(t, tc.wantFinal, entity.IsFinal())
			require.Equal(t, tc.wantAccess, entity.GetAccess())
			require.Equal(t, tc.wantResilient, entity.Resilient)
			if tc.wantDisplayName != "" {
				require.Equal(t, tc.wantDisplayName, entity.DisplayName)
			}
			if tc.wantDescription != "" {
				require.Equal(t, tc.wantDescription, entity.Description)
			}
		})
	}
}

func TestCollect_TypeHasSchemaAndSourceMap(t *testing.T) {
	ramlContent := entityRAML("types:\n  ObjectEntity:\n    (cti.cti): cti.x.y.object_entity.v1.0\n    type: object")

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	entity := reg.Types["cti.x.y.object_entity.v1.0"]
	require.NotNil(t, entity)
	require.NotNil(t, entity.Schema, "entity must have a JSON schema")
	require.NotEmpty(t, entity.Schema.Ref, "top-level schema must have a $ref to definitions")
	require.NotEmpty(t, entity.Schema.Definitions, "schema must have definitions")
	require.NotNil(t, entity.SourceMap, "entity must have a source map")
	require.Equal(t, "ObjectEntity", entity.SourceMap.Name)
}

// --- Multi-CTI tests ---

func TestCollect_MultiCTI(t *testing.T) {
	// A single RAML type can declare multiple CTI identifiers.
	// The collector must register the type under each declared CTI.
	ramlContent := entityRAML(`types:
  MultiCtiEntity:
    (cti.cti):
    - cti.x.y.multi_entity_1.v1.0
    - cti.x.y.multi_entity_2.v1.0
    type: object`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)
	require.Len(t, reg.Types, 2)
	require.Contains(t, reg.Types, "cti.x.y.multi_entity_1.v1.0")
	require.Contains(t, reg.Types, "cti.x.y.multi_entity_2.v1.0")
}

// --- CTI inheritance tests ---

func TestCollect_CTIInheritanceChain(t *testing.T) {
	// A child CTI type must inherit from the RAML type that carries the parent CTI.
	ramlContent := entityRAML(`types:
  ParentEntity:
    (cti.cti): cti.x.y.parent_entity.v1.0
    (cti.final): false
    type: object
  ChildEntity:
    (cti.cti): cti.x.y.parent_entity.v1.0~x.y.child_entity.v1.0
    type: ParentEntity`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)
	require.Contains(t, reg.Types, "cti.x.y.parent_entity.v1.0")
	require.Contains(t, reg.Types, "cti.x.y.parent_entity.v1.0~x.y.child_entity.v1.0")
	require.Len(t, reg.Types, 2)
}

func TestCollect_CTIChainViolation_NoParent(t *testing.T) {
	// A child CTI (containing "~") must have a RAML parent.
	// A plain "type: object" with no inheritance fails verification.
	ramlContent := entityRAML(`types:
  OrphanChild:
    (cti.cti): cti.x.y.parent_entity.v1.0~x.y.orphan.v1.0
    type: object`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	_, err := c.Collect()
	require.Error(t, err)
	require.ErrorContains(t, err, "has no parent")
}

func TestCollect_CTIChainViolation_WrongParent(t *testing.T) {
	// The child CTI says its parent is ParentA, but it inherits from ParentB.
	ramlContent := entityRAML(`types:
  ParentA:
    (cti.cti): cti.x.y.parent_a.v1.0
    (cti.final): false
    type: object
  ParentB:
    (cti.cti): cti.x.y.parent_b.v1.0
    (cti.final): false
    type: object
  ChildOfA:
    (cti.cti): cti.x.y.parent_a.v1.0~x.y.child.v1.0
    type: ParentB`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	_, err := c.Collect()
	require.Error(t, err)
	require.ErrorContains(t, err, "none of the parents has matching cti")
}

// --- CTI instance tests ---

func TestCollect_BasicInstance(t *testing.T) {
	ramlContent := entityRAML(`annotationTypes:
  Instances: SampleEntity[]

(Instances):
- id: cti.x.y.sample_entity.v1.0~x.y._.v1.0

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)
	require.Len(t, reg.Types, 1)
	require.Len(t, reg.Instances, 1)
	require.Contains(t, reg.Types, "cti.x.y.sample_entity.v1.0")
	require.Contains(t, reg.Instances, "cti.x.y.sample_entity.v1.0~x.y._.v1.0")

	instance := reg.Instances["cti.x.y.sample_entity.v1.0~x.y._.v1.0"]
	require.NotNil(t, instance)
	require.Equal(t, "cti.x.y.sample_entity.v1.0~x.y._.v1.0", instance.GetCTI())
	require.NotNil(t, instance.SourceMap)
}

func TestCollect_InstanceDisplayName(t *testing.T) {
	// Property annotated with (cti.display_name) provides the instance's display name.
	ramlContent := entityRAML(`annotationTypes:
  Instances: EntityWithName[]

(Instances):
- id: cti.x.y.entity_with_name.v1.0~x.y._.v1.0
  name: My Instance

types:
  EntityWithName:
    (cti.cti): cti.x.y.entity_with_name.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true
      name:
        type: string
        (cti.display_name): true`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	instance := reg.Instances["cti.x.y.entity_with_name.v1.0~x.y._.v1.0"]
	require.NotNil(t, instance)
	require.Equal(t, "My Instance", instance.DisplayName)
}

func TestCollect_InstanceDescription(t *testing.T) {
	// Property annotated with (cti.description) provides the instance's description.
	ramlContent := entityRAML(`annotationTypes:
  Instances: EntityWithDesc[]

(Instances):
- id: cti.x.y.entity_with_desc.v1.0~x.y._.v1.0
  desc: My Description

types:
  EntityWithDesc:
    (cti.cti): cti.x.y.entity_with_desc.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true
      desc:
        type: string
        (cti.description): true`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	instance := reg.Instances["cti.x.y.entity_with_desc.v1.0~x.y._.v1.0"]
	require.NotNil(t, instance)
	require.Equal(t, "My Description", instance.Description)
}

func TestCollect_InstanceAccessField(t *testing.T) {
	// Property annotated with (cti.access_field) provides the instance's access modifier.
	ramlContent := entityRAML(`annotationTypes:
  Instances: EntityWithAccess[]

(Instances):
- id: cti.x.y.entity_with_access.v1.0~x.y._.v1.0
  access: private

types:
  EntityWithAccess:
    (cti.cti): cti.x.y.entity_with_access.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true
      access:
        type: string
        (cti.access_field): true`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	instance := reg.Instances["cti.x.y.entity_with_access.v1.0~x.y._.v1.0"]
	require.NotNil(t, instance)
	require.Equal(t, consts.AccessModifierPrivate, instance.GetAccess())
}

func TestCollect_MultipleInstances(t *testing.T) {
	ramlContent := entityRAML(`annotationTypes:
  Instances: SampleEntity[]

(Instances):
- id: cti.x.y.sample_entity.v1.0~x.y.first.v1.0
- id: cti.x.y.sample_entity.v1.0~x.y.second.v1.0

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)
	require.Len(t, reg.Instances, 2)
	require.Contains(t, reg.Instances, "cti.x.y.sample_entity.v1.0~x.y.first.v1.0")
	require.Contains(t, reg.Instances, "cti.x.y.sample_entity.v1.0~x.y.second.v1.0")
}

// --- Error cases ---

func TestCollect_DuplicateCTIType(t *testing.T) {
	ramlContent := entityRAML(`types:
  Entity1:
    (cti.cti): cti.x.y.entity.v1.0
    type: object
  Entity2:
    (cti.cti): cti.x.y.entity.v1.0
    type: object`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	_, err := c.Collect()
	require.Error(t, err)
	require.ErrorContains(t, err, "duplicate cti.cti: cti.x.y.entity.v1.0")
}

func TestCollect_DuplicateInstance(t *testing.T) {
	ramlContent := entityRAML(`annotationTypes:
  Instances: SampleEntity[]

(Instances):
- id: cti.x.y.sample_entity.v1.0~x.y._.v1.0
- id: cti.x.y.sample_entity.v1.0~x.y._.v1.0

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	_, err := c.Collect()
	require.Error(t, err)
	require.ErrorContains(t, err, "duplicate cti entity")
}

func TestCollect_DuplicateAcrossTypeAndInstance(t *testing.T) {
	// A CTI that is both a type and an instance is also a duplicate.
	ramlContent := entityRAML(`annotationTypes:
  Instances: SampleEntity[]

(Instances):
- id: cti.x.y.sample_entity.v1.0~x.y.sub.v1.0

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true
  SubEntityType:
    (cti.cti): cti.x.y.sample_entity.v1.0~x.y.sub.v1.0
    type: SampleEntity`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	_, err := c.Collect()
	require.Error(t, err)
	require.ErrorContains(t, err, "duplicate cti entity")
}

// --- Schema pipeline behavior tests ---

func TestCollect_ArrayAnnotationMovedToItems(t *testing.T) {
	// postProcessCtiType / moveAnnotationsToArrayItem: (cti.schema) on an array
	// property must be moved from the array itself to its items schema.
	ramlContent := entityRAML(`types:
  EntityWithArraySchema:
    (cti.cti): cti.x.y.entity_with_array_schema.v1.0
    properties:
      schemas:
        type: object[]
        (cti.schema): cti.x.y.other_entity.v1.0`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	entity := reg.Types["cti.x.y.entity_with_array_schema.v1.0"]
	require.NotNil(t, entity)
	require.NotNil(t, entity.Schema)

	entityDef, ok := entity.Schema.Definitions["EntityWithArraySchema"]
	require.True(t, ok, "EntityWithArraySchema must be in schema definitions")
	require.NotNil(t, entityDef.Properties)

	arrayPropSchema, ok := entityDef.Properties.Get("schemas")
	require.True(t, ok, "schemas property must exist in the entity schema")

	// The (cti.schema) annotation must NOT remain on the array itself.
	require.Nil(t, arrayPropSchema.CTISchema,
		"(cti.schema) should have been moved from the array to its items")

	// The (cti.schema) annotation must have been moved to the array's item schema.
	require.NotNil(t, arrayPropSchema.Items,
		"array property must have an items schema")
	require.Equal(t, "cti.x.y.other_entity.v1.0", arrayPropSchema.Items.CTISchema,
		"(cti.schema) must be present on the items schema")
}

func TestCollect_ReferenceAnnotationMovedToArrayItems(t *testing.T) {
	// moveAnnotationsToArrayItem also handles (cti.reference) — same mechanism as (cti.schema).
	ramlContent := entityRAML(`types:
  EntityWithArrayRef:
    (cti.cti): cti.x.y.entity_with_array_ref.v1.0
    properties:
      refs:
        type: cti.CTI[]
        (cti.reference): cti.x.y.other_entity.v1.0`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	entity := reg.Types["cti.x.y.entity_with_array_ref.v1.0"]
	require.NotNil(t, entity)
	require.NotNil(t, entity.Schema)

	entityDef, ok := entity.Schema.Definitions["EntityWithArrayRef"]
	require.True(t, ok, "EntityWithArrayRef must be in schema definitions")
	require.NotNil(t, entityDef.Properties)

	arrayPropSchema, ok := entityDef.Properties.Get("refs")
	require.True(t, ok, "refs property must exist")
	require.Nil(t, arrayPropSchema.CTIReference,
		"(cti.reference) should have been moved from the array to its items")
	require.NotNil(t, arrayPropSchema.Items)
	require.Equal(t, "cti.x.y.other_entity.v1.0", arrayPropSchema.Items.CTIReference,
		"(cti.reference) must be present on the items schema")
}

func TestCollect_ImplicitSchemaOnCTITypedProperty(t *testing.T) {
	// insertImplicitSchema: a property typed as a CTI type gets an implicit
	// (cti.schema) annotation in the JSON schema pointing to that CTI type's identifier.
	ramlContent := entityRAML(`types:
  RefEntity:
    (cti.cti): cti.x.y.ref_entity.v1.0
    (cti.final): false
    type: object
  EntityWithRefProperty:
    (cti.cti): cti.x.y.entity_with_ref.v1.0
    properties:
      ref:
        type: RefEntity`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	entity := reg.Types["cti.x.y.entity_with_ref.v1.0"]
	require.NotNil(t, entity)
	require.NotNil(t, entity.Schema)

	entityDef, ok := entity.Schema.Definitions["EntityWithRefProperty"]
	require.True(t, ok, "EntityWithRefProperty must be in schema definitions")
	require.NotNil(t, entityDef.Properties)

	refPropSchema, ok := entityDef.Properties.Get("ref")
	require.True(t, ok, "ref property must exist in the entity schema")
	require.Equal(t, "cti.x.y.ref_entity.v1.0", refPropSchema.CTISchema,
		"ref property must have implicit (cti.schema) pointing to RefEntity")
}

func TestCollect_ExplicitSchemaAnnotationOnProperty(t *testing.T) {
	// An explicit (cti.schema) annotation on an object property is preserved in the JSON schema.
	ramlContent := entityRAML(`types:
  EntityWithSchema:
    (cti.cti): cti.x.y.entity_with_schema.v1.0
    properties:
      payload:
        type: object
        (cti.schema): cti.x.y.payload_type.v1.0`)

	c := newTestCollector(t, map[string]string{"entities.raml": ramlContent})
	reg, err := c.Collect()
	require.NoError(t, err)

	entity := reg.Types["cti.x.y.entity_with_schema.v1.0"]
	require.NotNil(t, entity)
	require.NotNil(t, entity.Schema)

	entityDef, ok := entity.Schema.Definitions["EntityWithSchema"]
	require.True(t, ok)
	require.NotNil(t, entityDef.Properties)

	payloadSchema, ok := entityDef.Properties.Get("payload")
	require.True(t, ok)
	require.Equal(t, "cti.x.y.payload_type.v1.0", payloadSchema.CTISchema)
}
