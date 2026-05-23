package ctipackage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/metadata/testsupp"
)

func newExamplePkg(t *testing.T, name string, tc testsupp.PackageTestCase, examples []string) *Package {
	t.Helper()
	testDir := testsupp.InitTestPackageFiles(t, name, tc)
	pkg, err := New(testDir,
		WithRamlxVersion("1.0"),
		WithID(tc.PkgId),
		WithEntities(tc.Entities),
		WithExamples(examples),
	)
	require.NoError(t, err)
	require.NoError(t, pkg.Initialize())
	require.NoError(t, pkg.Read())
	return pkg
}

func Test_ParseExamples_Empty(t *testing.T) {
	pkg := newExamplePkg(t, "examples_empty", testsupp.PackageTestCase{
		PkgId: "x.y",
		Files: map[string]string{},
	}, nil)

	require.NoError(t, pkg.Parse())
	require.NoError(t, pkg.ParseExamples())
	require.Nil(t, pkg.ExamplesRegistry)
}

func Test_ParseExamples_RAML(t *testing.T) {
	tc := testsupp.PackageTestCase{
		PkgId:    "x.y",
		Entities: []string{"entities/entity.raml"},
		Files: map[string]string{
			"entities/entity.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    (cti.final): false
    properties:
      name: string
`),
			"examples/example.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  SampleEntityExample:
    (cti.cti): cti.x.y.sample_entity_example.v1.0
    (cti.final): false
    properties:
      name: string
`),
		},
	}

	pkg := newExamplePkg(t, "examples_raml", tc, []string{"examples/example.raml"})

	require.NoError(t, pkg.ParseExamples())
	require.NotNil(t, pkg.ExamplesRegistry)
	require.Len(t, pkg.ExamplesRegistry.Index, 1)
	require.Contains(t, pkg.ExamplesRegistry.Types, "cti.x.y.sample_entity_example.v1.0")
}

func Test_ParseExamples_YAML(t *testing.T) {
	tc := testsupp.PackageTestCase{
		PkgId: "x.y",
		Files: map[string]string{
			"examples/example_type.yaml": strings.TrimSpace(`
#%CTI Type 1.0
cti: cti.x.y.yaml_example.v1.0
final: false
access: public
schema:
  $schema: http://json-schema.org/draft-07/schema#
  type: object
  properties:
    name:
      type: string
`),
		},
	}

	pkg := newExamplePkg(t, "examples_yaml", tc, []string{"examples/example_type.yaml"})

	require.NoError(t, pkg.ParseExamples())
	require.NotNil(t, pkg.ExamplesRegistry)
	require.Len(t, pkg.ExamplesRegistry.Index, 1)
	require.Contains(t, pkg.ExamplesRegistry.Types, "cti.x.y.yaml_example.v1.0")
}

func Test_ParseExamples_CollisionWithMainEntity(t *testing.T) {
	tc := testsupp.PackageTestCase{
		PkgId:    "x.y",
		Entities: []string{"entities/entity.raml"},
		Files: map[string]string{
			"entities/entity.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    (cti.final): false
    properties:
      name: string
`),
			// Example reuses the same CTI expression as the main entity.
			"examples/example.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    (cti.final): false
    properties:
      name: string
`),
		},
	}

	pkg := newExamplePkg(t, "examples_collision", tc, []string{"examples/example.raml"})

	err := pkg.ParseExamples()
	require.Error(t, err)
	require.Contains(t, err.Error(), "collides")
}

// Test_ValidateExamples_Valid tests the canonical example pattern from real packages:
// the entity file (in "entities") defines both a CTI type and a library-level annotation type,
// and the example file (in "examples") imports that annotation type and uses it to create
// concrete instances. This matches patterns seen in real packages (e.g. wr/examples, tm/examples).
func Test_ValidateExamples_Valid(t *testing.T) {
	tc := testsupp.PackageTestCase{
		PkgId:    "x.y",
		Entities: []string{"entities/entity.raml"},
		Files: map[string]string{
			// Entity file defines both the CTI type and the annotation type used to create instances.
			"entities/entity.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

annotationTypes:
  SampleEntities:
    type: SampleEntity[]
    allowedTargets: [Library]

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    (cti.final): false
    properties:
      id:
        (cti.id): true
      name: string
`),
			// Example file imports the entity file and uses its annotation type to create
			// a concrete instance that refers back to the entity type in the main package.
			// The relative path resolves from examples/ to the entity file at the package root.
			"examples/example.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml
  entities: ../entities/entity.raml

(entities.SampleEntities):
  - id: cti.x.y.sample_entity.v1.0~x.y.example_data.v1.0
    name: Example Name
`),
		},
	}

	pkg := newExamplePkg(t, "examples_validate_valid", tc, []string{"examples/example.raml"})

	require.NoError(t, pkg.ValidateExamples())
	require.NotNil(t, pkg.ExamplesRegistry)
	require.Len(t, pkg.ExamplesRegistry.Index, 1)
	require.Contains(t, pkg.ExamplesRegistry.Instances, "cti.x.y.sample_entity.v1.0~x.y.example_data.v1.0")
}

func Test_ValidateExamples_Empty(t *testing.T) {
	pkg := newExamplePkg(t, "examples_validate_empty", testsupp.PackageTestCase{
		PkgId: "x.y",
		Files: map[string]string{},
	}, nil)

	require.NoError(t, pkg.Parse())
	require.NoError(t, pkg.ValidateExamples())
}
