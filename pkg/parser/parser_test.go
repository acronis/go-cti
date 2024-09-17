package parser_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-cti/pkg/parser"
)

func getAbsPath(path string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, path)
}

func TestParsePackageAbsPath(t *testing.T) {
	path := getAbsPath("./fixtures/valid/empty/index.json")
	p, err := parser.ParsePackage(path)
	require.NoError(t, err)
	require.NotNil(t, p)

	baseDir := filepath.Dir(path)

	require.Equal(t, baseDir, p.BaseDir)
	require.NotNil(t, p.Registry)
	require.Empty(t, p.Registry.Total)
	require.NotNil(t, p.RAML)
}

func TestParsePackageRelPath(t *testing.T) {
	path := "./fixtures/valid/empty/index.json"
	p, err := parser.ParsePackage(path)
	require.NoError(t, err)
	require.NotNil(t, p)

	absPath := getAbsPath(path)
	baseDir := filepath.Dir(absPath)

	require.Equal(t, baseDir, p.BaseDir)
	require.NotNil(t, p.Registry)
	require.Empty(t, p.Registry.Total)
	require.NotNil(t, p.RAML)
}

func TestParsePackageInvalidIdxType(t *testing.T) {
	path := "./fixtures/invalid/empty/index.json"
	_, err := parser.ParsePackage(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid index file type: abc.v1.0")
}

func TestParsePackageMissingFile(t *testing.T) {
	path := "./fixtures/invalid/missing_files/index.json"
	_, err := parser.ParsePackage(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non_existent_file.raml: The system cannot find the file specified.")
}

func TestParsePackageDuplicateType(t *testing.T) {
	path := "./fixtures/invalid/duplicate_type/index.json"
	_, err := parser.ParsePackage(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate cti.cti: cti.a.p.unique_entity.v1.0")
}

func TestParsePackageDuplicateInstance(t *testing.T) {
	path := "./fixtures/invalid/duplicate_instance/index.json"
	_, err := parser.ParsePackage(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate cti entity cti.a.p.sample_entity.v1.0~a.p._.v1.0")
}

func TestParsePackageDuplicateTypeInstance(t *testing.T) {
	path := "./fixtures/invalid/duplicate_type_instance/index.json"
	_, err := parser.ParsePackage(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate cti entity cti.a.p.sample_entity.v1.0~a.p._.v1.0")
}

func generateGoldenFiles(t *testing.T, baseDir string, collections map[string]cti.Entities) {
	for fragmentPath, entities := range collections {
		path := filepath.Join(baseDir, fragmentPath)
		goldenPath := strings.TrimSuffix(path, filepath.Ext(path)) + "_golden.json"
		err := func() error {
			f, err := os.OpenFile(goldenPath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("failed to open golden file %s: %v", goldenPath, err)
			}
			defer f.Close()
			stat, err := f.Stat()
			if err != nil {
				return fmt.Errorf("Error getting file info: %v\n", err)
			}

			if stat.Size() == 0 {
				bytes, err := json.MarshalIndent(entities, "", "  ")
				_, err = f.Write(bytes)
				return err
			} else {
				d := json.NewDecoder(f)
				var golden []*cti.EntityStructured
				err = d.Decode(&golden)
				require.NoError(t, err)
				var source []*cti.EntityStructured
				for _, entity := range entities {
					bytes, err := json.Marshal(entity)
					require.NoError(t, err)
					var structuredEntity *cti.EntityStructured
					err = json.Unmarshal(bytes, &structuredEntity)
					require.NoError(t, err)
					source = append(source, structuredEntity)
				}
				found := false
				for _, entity := range golden {
					for _, sourceEntity := range source {
						if entity.Cti == sourceEntity.Cti {
							found = true
							require.Equal(t, entity, sourceEntity)
							break
						}
					}
				}
				require.True(t, found, "Failed to find corresponding CTI entity in %s", goldenPath)
				return nil
			}
		}()
		require.NoError(t, err)
	}
}

func TestParseAnnotations(t *testing.T) {
	path := "./fixtures/valid/annotations/index.json"

	p, err := parser.ParsePackage(path)
	require.NoError(t, err)
	generateGoldenFiles(t, p.BaseDir, p.Registry.FragmentEntities)

	require.Equal(t, 26, len(p.Registry.Total))
	require.Equal(t, 22, len(p.Registry.Types))
	require.Equal(t, 4, len(p.Registry.Instances))
}
