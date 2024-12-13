package ctipackage

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/testsupp"
	"github.com/acronis/go-stacktrace"
	slogex "github.com/acronis/go-stacktrace/slogex"
)

func generateGoldenFiles(t *testing.T, baseDir string, collections map[string]metadata.Entities) {
	t.Helper()

	for fragmentPath, entities := range collections {
		path := filepath.Join(baseDir, fragmentPath)
		goldenPath := strings.TrimSuffix(path, filepath.Ext(path)) + "_golden.json"
		err := func() error {
			f, err := os.OpenFile(goldenPath, os.O_RDWR|os.O_CREATE, 0644)
			require.NoError(t, err)

			defer f.Close()
			stat, err := f.Stat()
			require.NoError(t, err)

			if stat.Size() == 0 {
				bytes, err := json.MarshalIndent(entities, "", "  ")
				require.NoError(t, err)
				_, err = f.Write(bytes)
				return err
			}

			d := json.NewDecoder(f)
			var golden []*metadata.EntityStructured
			require.NoError(t, d.Decode(&golden))

			var source []*metadata.EntityStructured
			for _, entity := range entities {
				bytes, err := json.Marshal(entity)
				require.NoError(t, err)

				var structuredEntity *metadata.EntityStructured
				require.NoError(t, json.Unmarshal(bytes, &structuredEntity))
				source = append(source, structuredEntity)
			}
			found := func() bool {
				for _, entity := range golden {
					for _, sourceEntity := range source {
						if entity.Cti == sourceEntity.Cti {
							require.Equal(t, entity, sourceEntity)
							return true
						}
					}
				}
				return false
			}()

			require.True(t, found, "Failed to find corresponding CTI entity in %s", goldenPath)
			return nil

		}()
		require.NoError(t, err)
	}
}

func Test_ParseAnnotations(t *testing.T) {
	testsupp.InitLog(t)

	type testCase struct {
		parserTestCase

		total     int
		types     int
		instances int
	}

	testCases := []testCase{
		{
			parserTestCase: parserTestCase{
				name:  "annotations",
				pkgId: "x.y",
				entities: []string{
					"entities/cti.raml",
					"entities/final.raml",
					"entities/id.raml",
					"entities/display_name.raml",
					"entities/description.raml",
					"entities/asset.raml",
					"entities/overridable.raml",
					"entities/reference.raml",
					"entities/schema.raml",
				},
				files: map[string]string{
					"entities/asset.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

annotationTypes:
  Instances: EntityWithAsset[]

(Instances):
- id: cti.x.y.entity_with_asset.v1.0~x.y._.v1.0
  asset: assets/asset.txt

types:
  EntityWithAsset:
    (cti.cti): cti.x.y.entity_with_asset.v1.0
    properties:
      id:
        (cti.id): true
      asset:
        (cti.asset): true
`),
					"assets/asset.txt": "Sample text",
					"entities/cti.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      name: string
      age: number
  OtherEntity:
    (cti.cti): cti.x.y.other_entity.v1.0
    properties:
      value: integer
  SampleDerivedEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0~x.y._.v1.0
    type: object
  MultiCtiEntity:
    (cti.cti):
    - cti.x.y.multi_cti_entity_1.v1.0
    - cti.x.y.multi_cti_entity_2.v1.0
    type: object
`),
					"entities/description.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

annotationTypes:
  InstancesWithDescription: EntityWithDescription[]

(InstancesWithDescription):
- id: cti.x.y.entity_with_description.v1.0~x.y._.v1.0
  description: Instance Description

types:
  EntityWithDescription:
    (cti.cti): cti.x.y.entity_with_description.v1.0
    properties:
      id:
        (cti.id): true
      description:
        (cti.description): true
`),
					"entities/display_name.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

annotationTypes:
  InstancesWithDisplayName: EntityWithDisplayName[]

(InstancesWithDisplayName):
- id: cti.x.y.entity_with_display_name.v1.0~x.y._.v1.0
  name: Instance Name

types:
  EntityWithDisplayName:
    (cti.cti): cti.x.y.entity_with_display_name.v1.0
    properties:
      id:
        (cti.id): true
      name:
        (cti.display_name): true
`),
					"entities/final.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  NonFinalEntity:
    (cti.final): false
    (cti.cti): cti.x.y.non_final_entity.v1.0
    type: object
  FinalEntity:
    (cti.cti): cti.x.y.non_final_entity.v1.0~x.y._.v1.0
    type: object
`),
					"entities/id.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

annotationTypes:
  Instances: EntityWithInstance[]

(Instances):
- id: cti.x.y.entity_with_instance.v1.0~x.y._.v1.0

types:
  EntityWithInstance:
    (cti.cti): cti.x.y.entity_with_instance.v1.0
    properties:
      id:
        (cti.id): true
`),
					"entities/overridable.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  EntityWithOverridable:
    (cti.cti): cti.x.y.entity_with_overridable.v1.0
    (cti.overridable): true
    properties:
      overridable:
        (cti.overridable): true
      non_overridable:
`),
					"entities/reference.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  EntityWithReference:
    (cti.cti): cti.x.y.entity_with_reference.v1.0
    properties:
      implicit_reference:
        type: cti.CTI
        (cti.reference): true
      single_reference:
        type: cti.CTI
        (cti.reference): cti.x.y.other_entity.v1.0
      multiple_references:
        type: cti.CTI
        (cti.reference):
        - cti.x.y.other_entity.v1.0
        - cti.x.y.sample_entity.v1.0
  EntityWithArrayReference:
    (cti.cti): cti.x.y.entity_with_array_reference.v1.0
    properties:
      array_reference:
        type: cti.CTI[]
        (cti.reference): cti.x.y.other_entity.v1.0
      array_references:
        type: cti.CTI[]
        (cti.reference):
        - cti.x.y.other_entity.v1.0
        - cti.x.y.sample_entity.v1.0
`),
					"entities/schema.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: ../.ramlx/cti.raml

types:
  EntityWithSchema:
    (cti.cti): cti.x.y.entity_with_schema.v1.0
    properties:
      single_schema:
        (cti.schema): cti.x.y.sample_entity.v1.0
      multi_schema:
        (cti.schema):
        - cti.x.y.other_entity.v1.0
        - cti.x.y.sample_entity.v1.0
  EntityWithSchemaNestedAnnotations:
    (cti.cti): cti.x.y.entity_with_schema_nested_annotations.v1.0
    properties:
      schema:
        (cti.schema): cti.x.y.entity_with_asset.v1.0
  EntityWithSchemaNestedSchema:
    (cti.cti): cti.x.y.entity_with_schema_nested_schema.v1.0
    properties:
      schema:
        (cti.schema): cti.x.y.entity_with_schema_nested_annotations.v1.0
  EntityWithArraySchema:
    (cti.cti): cti.x.y.entity_with_array_schema.v1.0
    properties:
      schema:
        type: object[]
        (cti.schema): cti.x.y.entity_with_schema_nested_annotations.v1.0
  EntityWithRecursiveSchema:
    (cti.cti): cti.x.y.entity_with_recursive_schema.v1.0
    properties:
      schema:
        type: object
        (cti.schema): cti.x.y.entity_with_recursive_schema.v1.0
`),
				},
			},
			total:     23,
			types:     19,
			instances: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pkg, err := New(initParseTest(t, tc.parserTestCase),
				WithRamlxVersion("1.0"),
				WithID(tc.pkgId),
				WithEntities(tc.entities))

			require.NoError(t, err)
			require.NoError(t, pkg.Initialize())
			require.NoError(t, pkg.Read())

			if err := pkg.Parse(); err != nil {
				slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
				require.Error(t, err)
			}

			generateGoldenFiles(t, pkg.BaseDir, pkg.LocalRegistry.FragmentEntities)

			require.EqualValues(t, tc.total, len(pkg.LocalRegistry.Index))
			require.EqualValues(t, tc.types, len(pkg.LocalRegistry.Types))
			require.EqualValues(t, tc.instances, len(pkg.LocalRegistry.Instances))
		})
	}
}
