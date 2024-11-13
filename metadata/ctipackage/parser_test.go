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

type parserTestCase struct {
	name     string
	pkgId    string
	entities []string
	files    map[string]string
}

func Test_EmptyPackage(t *testing.T) {
	pkg, err := New("./fixtures/valid/empty")
	require.NoError(t, err)
	require.NoError(t, pkg.Read())
	require.NoError(t, pkg.Parse())

	require.NotNil(t, pkg.Registry)
	require.Empty(t, pkg.Registry.Total)
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

func initParseTest(t *testing.T, tc parserTestCase) string {
	t.Helper()

	testDir := filepath.Join("./testdata", strings.ReplaceAll(tc.name, " ", "_"))
	require.NoError(t, os.RemoveAll(testDir))
	require.NoError(t, os.MkdirAll(testDir, os.ModePerm))

	for name, content := range tc.files {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(testDir, name)), os.ModePerm))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, name), []byte(content), os.ModePerm))
	}

	return testDir
}

func Test_InvalidPackage(t *testing.T) {
	testsupp.InitLog(t)

	type testCase struct {
		parserTestCase
		expectedError string
	}

	testCases := []testCase{
		{
			parserTestCase: parserTestCase{
				name:     "duplicate type",
				pkgId:    "x.y",
				entities: []string{"entities.raml"},
				files: map[string]string{"entities.raml": strings.TrimSpace(`
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
			parserTestCase: parserTestCase{
				name:     "duplicate instance",
				pkgId:    "x.y",
				entities: []string{"entities.raml"},
				files: map[string]string{"entities.raml": strings.TrimSpace(`
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
			parserTestCase: parserTestCase{
				name:     "duplicate type instance",
				pkgId:    "x.y",
				entities: []string{"entities.raml"},
				files: map[string]string{"entities.raml": strings.TrimSpace(`
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
			parserTestCase: parserTestCase{
				name:     "missing file",
				pkgId:    "x.y",
				entities: []string{"non_existent_file.raml"},
			},
			expectedError: "non_existent_file.raml: The system cannot find the file specified.",
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

			{
				err := pkg.Parse()
				require.Error(t, err)
				require.ErrorContains(t, err, tc.expectedError)

				slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
			}
		})
	}
}
