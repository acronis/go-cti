package ctipackage

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/metadata/testsupp"
	"github.com/acronis/go-stacktrace"
	slogex "github.com/acronis/go-stacktrace/slogex"
)

func Test_EmptyPackage(t *testing.T) {
	testPath := "./testdata/valid/empty"

	require.NoError(t, os.RemoveAll(testPath))
	require.NoError(t, os.MkdirAll(testPath, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(testPath, "index.json"), []byte(`{"package_id": "test.pkg"}`), os.ModePerm))

	pkg, err := New(testPath)
	require.NoError(t, err)
	require.NoError(t, pkg.Read())
	require.NoError(t, pkg.Parse())

	require.NotNil(t, pkg.LocalRegistry)
	require.Empty(t, pkg.LocalRegistry.Index)
}

func Test_EmptyIndex(t *testing.T) {
	testPath := "./testdata/invalid/empty"

	require.NoError(t, os.RemoveAll(testPath))
	require.NoError(t, os.MkdirAll(testPath, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(testPath, "index.json"), []byte(`{}`), os.ModePerm))

	pkg, err := New(testPath)
	require.NoError(t, err)
	require.ErrorContains(t, pkg.Read(), "invalid package ID")
}

func Test_GenerateIndexRaml(t *testing.T) {
	tests := map[string]struct {
		pkg             Package
		includeExamples bool
		expectedOutput  string
	}{
		"no examples": {
			pkg: Package{
				Index: &Index{
					Entities: []string{"entity1.raml", "entity2.raml"},
					Examples: []string{"example1.raml"},
				},
			},
			includeExamples: false,
			expectedOutput:  "#%RAML 1.0 Library\nuses:\n  e1: entity1.raml\n  e2: entity2.raml",
		},
		"with examples": {
			pkg: Package{
				Index: &Index{
					Entities: []string{"entity1.raml"},
					Examples: []string{"example1.raml", "example2.raml"},
				},
			},
			includeExamples: true,
			expectedOutput:  "#%RAML 1.0 Library\nuses:\n  e1: entity1.raml\n  x1: example1.raml\n  x2: example2.raml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output := tt.pkg.generateRAML(tt.includeExamples)
			require.Equal(t, tt.expectedOutput, output)
		})
	}
}

func Test_ValidPackage(t *testing.T) {
	testsupp.InitLog(t)

	testCases := map[string]testsupp.PackageTestCase{
		"valid CTI types": {
			PkgId:    "x.y",
			Entities: []string{"entities.raml", "instance.yaml"},
			Files: map[string]string{"entities.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  ObjectEntity:
    (cti.cti): cti.x.y.entity_object.v1.0
    type: object

  ScalarEntity:
    (cti.cti): cti.x.y.entity_scalar.v1.0
    type: string

  NilEntity:
    (cti.cti): cti.x.y.entity_nil.v1.0
    type: nil
`),
				"instance.yaml": strings.TrimSpace(`
#%CTI Instance 1.0

cti: cti.x.y.entity_scalar.v1.0~x.y.instance.v1.0
final: true
access: protected
resilient: false
display_name: My Instance
values: my_value
`)},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			pkg, err := New(testsupp.InitTestPackageFiles(t, name, tc),
				WithRamlxVersion("1.0"),
				WithID(tc.PkgId),
				WithEntities(tc.Entities))

			require.NoError(t, err)
			require.NoError(t, pkg.Initialize())
			require.NoError(t, pkg.Read())

			{
				err := pkg.Parse()
				if err != nil {
					slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
					t.Fatalf("unexpected error: %v", err)
				}
				err = pkg.Validate()
				if err != nil {
					slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func Test_InvalidPackage(t *testing.T) {
	testsupp.InitLog(t)

	testCases := map[string]struct {
		testsupp.PackageTestCase
		expectedError string
	}{
		"duplicate type": {
			PackageTestCase: testsupp.PackageTestCase{
				PkgId:    "x.y",
				Entities: []string{"entities.raml"},
				Files: map[string]string{"entities.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  UniqueEntity:
    (cti.cti): cti.x.y.unique_entity.v1.0
    type: object

  DuplicateEntity:
    (cti.cti): cti.x.y.unique_entity.v1.0
    type: object
`)},
			},
			expectedError: "duplicate cti.cti: cti.x.y.unique_entity.v1.0",
		},
		"duplicate instance": {
			PackageTestCase: testsupp.PackageTestCase{
				PkgId:    "x.y",
				Entities: []string{"entities.raml"},
				Files: map[string]string{"entities.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

annotationTypes:
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
        (cti.id): true
`)},
			},
			expectedError: "duplicate cti entity cti.x.y.sample_entity.v1.0~x.y._.v1.0",
		},
		"duplicate type instance": {
			PackageTestCase: testsupp.PackageTestCase{
				PkgId:    "x.y",
				Entities: []string{"entities.raml"},
				Files: map[string]string{"entities.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

annotationTypes:
  Instances: SampleEntity[]

(Instances):
- id: cti.x.y.sample_entity.v1.0~x.y._.v1.0

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      id:
        type: cti.CTI
        (cti.id): true

  TypeWithInstanceCti:
    (cti.cti): cti.x.y.sample_entity.v1.0~x.y._.v1.0
    type: SampleEntity
`) + "\n"},
			},
			expectedError: "duplicate cti entity cti.x.y.sample_entity.v1.0~x.y._.v1.0",
		},
		"missing file": {
			PackageTestCase: testsupp.PackageTestCase{
				PkgId:    "x.y",
				Entities: []string{"non_existent_file.raml"},
			},
			// Win error: non_existent_file.raml: no such file or directory
			// POSIX error: non_existent_file.raml: The system cannot find the file specified.
			expectedError: "non_existent_file.raml",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			pkg, err := New(testsupp.InitTestPackageFiles(t, name, tc.PackageTestCase),
				WithRamlxVersion("1.0"),
				WithID(tc.PkgId),
				WithEntities(tc.Entities))

			require.NoError(t, err)
			require.NoError(t, pkg.Initialize())
			require.NoError(t, pkg.Read())

			{
				err := pkg.Parse()
				require.Error(t, err)
				require.ErrorContains(t, err, tc.expectedError)

				slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
			}
		})
	}
}
