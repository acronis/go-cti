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
	require.ErrorContains(t, pkg.Read(), "read index file: check index file: package id is missing")
}

func Test_InvalidPackage(t *testing.T) {
	testsupp.InitLog(t)

	type testCase struct {
		testsupp.PackageTestCase
		expectedError string
	}

	testCases := []testCase{
		{
			PackageTestCase: testsupp.PackageTestCase{
				Name:     "duplicate type",
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
		{
			PackageTestCase: testsupp.PackageTestCase{
				Name:     "duplicate instance",
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
		{
			PackageTestCase: testsupp.PackageTestCase{
				Name:     "duplicate type instance",
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
    type: object
`) + "\n"},
			},
			expectedError: "duplicate cti entity cti.x.y.sample_entity.v1.0~x.y._.v1.0",
		},
		{
			PackageTestCase: testsupp.PackageTestCase{
				Name:     "missing file",
				PkgId:    "x.y",
				Entities: []string{"non_existent_file.raml"},
			},
			// Win error: non_existent_file.raml: no such file or directory
			// POSIX error: non_existent_file.raml: The system cannot find the file specified.
			expectedError: "non_existent_file.raml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			pkg, err := New(testsupp.InitTestPackageFiles(t, tc.PackageTestCase),
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
