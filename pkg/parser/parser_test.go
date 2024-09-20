package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/pkg/cti"
)

func getAbsPath(path string) string {
	p, _ := filepath.Abs(path)
	return p
}

func Test_ParsePackageAbsPath(t *testing.T) {
	path := getAbsPath("./fixtures/valid/empty/index.json")
	pp, err := ParsePackage(path)
	p := pp.(*parserImpl)
	require.NoError(t, err)
	require.NotNil(t, p)

	baseDir := filepath.Dir(path)

	require.Equal(t, baseDir, p.BaseDir)
	require.NotNil(t, p.Registry)
	require.Empty(t, p.Registry.Total)
	require.NotNil(t, p.RAML)
}

func Test_ParsePackageRelPath(t *testing.T) {
	path := "./fixtures/valid/empty/index.json"
	pp, err := ParsePackage(path)
	p := pp.(*parserImpl)
	require.NoError(t, err)
	require.NotNil(t, p)

	absPath := getAbsPath(path)
	baseDir := filepath.Dir(absPath)

	require.Equal(t, baseDir, p.BaseDir)
	require.NotNil(t, p.Registry)
	require.Empty(t, p.Registry.Total)
	require.NotNil(t, p.RAML)
}

func Test_Invalid(t *testing.T) {
	type testCase struct {
		name          string
		fixturePath   string
		expectedError string
	}

	testCases := []testCase{
		{
			name:          "empty package",
			fixturePath:   "./fixtures/invalid/empty/index.json",
			expectedError: "error checking index file: missing app code",
		},
		{
			name:          "missing file",
			fixturePath:   "./fixtures/invalid/missing_files/index.json",
			expectedError: "non_existent_file.raml: The system cannot find the file specified.",
		},
		{
			name:          "duplicate type",
			fixturePath:   "./fixtures/invalid/duplicate_type/index.json",
			expectedError: "duplicate cti.cti: cti.a.p.unique_entity.v1.0",
		},
		{
			name:          "duplicate instance",
			fixturePath:   "./fixtures/invalid/duplicate_instance/index.json",
			expectedError: "duplicate cti entity cti.a.p.sample_entity.v1.0~a.p._.v1.0",
		},
		{
			name:          "duplicate type instance",
			fixturePath:   "./fixtures/invalid/duplicate_type_instance/index.json",
			expectedError: "duplicate cti entity cti.a.p.sample_entity.v1.0~a.p._.v1.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePackage(getAbsPath(tc.fixturePath))
			require.Error(t, err)
			require.ErrorContains(t, err, tc.expectedError)
		})
	}

}

func generateGoldenFiles(t *testing.T, baseDir string, collections map[string]cti.Entities) {
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
			var golden []*cti.EntityStructured
			require.NoError(t, d.Decode(&golden))

			var source []*cti.EntityStructured
			for _, entity := range entities {
				bytes, err := json.Marshal(entity)
				require.NoError(t, err)

				var structuredEntity *cti.EntityStructured
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
	path := "./fixtures/valid/annotations/index.json"

	pp, err := ParsePackage(path)
	p := pp.(*parserImpl)
	require.NoError(t, err)
	generateGoldenFiles(t, p.BaseDir, p.Registry.FragmentEntities)

	require.Equal(t, 26, len(p.Registry.Total))
	require.Equal(t, 22, len(p.Registry.Types))
	require.Equal(t, 4, len(p.Registry.Instances))
}
