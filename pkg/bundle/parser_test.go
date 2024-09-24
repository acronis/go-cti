package bundle

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/pkg/testsupp"
	"github.com/acronis/go-stacktrace"
)

type parserTestCase struct {
	name     string
	appCode  string
	entities []string
	files    map[string]string
}

func Test_EmptyPackage(t *testing.T) {
	bd := New("./fixtures/valid/empty")
	require.NoError(t, bd.Read())
	require.NoError(t, bd.Parse())

	require.NotNil(t, bd.Registry)
	require.Empty(t, bd.Registry.Total)
}

func Test_EmptyIndex(t *testing.T) {
	testPath := "./testdata/invalid/empty"

	require.NoError(t, os.RemoveAll(testPath))
	require.NoError(t, os.MkdirAll(testPath, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(testPath, "index.json"), []byte(`{}`), os.ModePerm))

	bd := New(testPath)
	require.ErrorContains(t, bd.Read(), "read index file: check index file: missing app code")
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

func Test_InvalidBundle(t *testing.T) {
	testsupp.InitLog(t)

	type testCase struct {
		parserTestCase
		expectedError string
	}

	testCases := []testCase{
		{
			parserTestCase: parserTestCase{
				name:     "duplicate type",
				appCode:  "x.y",
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
				appCode:  "x.y",
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
				appCode:  "x.y",
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
				appCode:  "x.y",
				entities: []string{"non_existent_file.raml"},
			},
			expectedError: "non_existent_file.raml: The system cannot find the file specified.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			bd := New(initParseTest(t, tc.parserTestCase),
				WithRamlxVersion("1.0"),
				WithAppCode(tc.appCode),
				WithEntities(tc.entities))

			require.NoError(t, bd.Initialize())
			require.NoError(t, bd.Read())

			err := bd.Parse()
			require.Error(t, err)

			slog.Error("Command failed", stacktrace.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))

			require.ErrorContains(t, bd.Parse(), tc.expectedError)
		})
	}
}
