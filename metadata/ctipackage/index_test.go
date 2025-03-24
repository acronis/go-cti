package ctipackage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func initIndexFixture(t *testing.T, content []byte) {
	testPath := filepath.Join("testdata", "indexes")
	indexPath := filepath.Join(testPath, "index.json")
	require.NoError(t, os.RemoveAll(testPath))
	require.NoError(t, os.MkdirAll(testPath, os.ModePerm))
	require.NoError(t, os.WriteFile(indexPath, content, os.ModePerm))
}

func Test_ReadIndexFile(t *testing.T) {
	validIndexContent := []byte(`{
		"package_id": "test.pkg",
		"ramlx_version": "1.0",
		"apis": ["api.raml"],
		"entities": ["entity.raml"],
		"assets": ["asset.png"],
		"dictionaries": ["dict.json"],
		"depends": {"dep": "1.0"},
		"examples": ["example.raml"],
		"serialized": [".cache.json"]
	}`)

	invalidIndexContent := []byte(`{
		"package_id": "",
		"ramlx_version": "1.0",
		"apis": ["api.raml"],
		"entities": ["entity.raml"],
		"assets": ["asset.png"],
		"dictionaries": ["dict.json"],
		"depends": {"dep": "1.0"},
		"examples": ["example.raml"],
		"serialized": [".cache.json"]
	}`)

	tests := []struct {
		name        string
		content     []byte
		expectError bool
	}{
		{"ValidIndexFile", validIndexContent, false},
		{"InvalidIndexFile", invalidIndexContent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initIndexFixture(t, tt.content)
			idx, err := ReadIndexFile(filepath.Join("testdata", "indexes", "index.json"))
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, idx)
				require.Equal(t, "test.pkg", idx.PackageID)
			}
		})
	}
}

func Test_IndexCheck(t *testing.T) {
	tests := []struct {
		name        string
		index       Index
		expectError bool
	}{
		{
			name: "ValidIndex",
			index: Index{
				PackageID: "test.pkg",
				Apis:      []string{"api1.raml"},
				Entities:  []string{"entity1.raml"},
				Examples:  []string{"example1.raml"},
			},
			expectError: false,
		},
		{
			name: "EmptyApiPath",
			index: Index{
				PackageID: "test.pkg",
				Apis:      []string{""},
				Entities:  []string{"entity1.raml"},
				Examples:  []string{"example1.raml"},
			},
			expectError: true,
		},
		{
			name: "InvalidApiExtension",
			index: Index{
				PackageID: "test.pkg",
				Apis:      []string{"api1.txt"},
				Entities:  []string{"entity1.raml"},
				Examples:  []string{"example1.raml"},
			},
			expectError: true,
		},
		{
			name: "EmptyEntityPath",
			index: Index{
				PackageID: "test.pkg",
				Apis:      []string{"api1.raml"},
				Entities:  []string{""},
				Examples:  []string{"example1.raml"},
			},
			expectError: true,
		},
		{
			name: "InvalidEntityExtension",
			index: Index{
				PackageID: "test.pkg",
				Apis:      []string{"api1.raml"},
				Entities:  []string{"entity1.txt"},
				Examples:  []string{"example1.raml"},
			},
			expectError: true,
		},
		{
			name: "EmptyExamplePath",
			index: Index{
				PackageID: "test.pkg",
				Apis:      []string{"api1.raml"},
				Entities:  []string{"entity1.raml"},
				Examples:  []string{""},
			},
			expectError: true,
		},
		{
			name: "InvalidExampleExtension",
			index: Index{
				PackageID: "test.pkg",
				Apis:      []string{"api1.raml"},
				Entities:  []string{"entity1.raml"},
				Examples:  []string{"example1.txt"},
			},
			expectError: true,
		},
		{
			name: "MissingPackageID",
			index: Index{
				PackageID: "",
				Apis:      []string{"api1.raml"},
				Entities:  []string{"entity1.raml"},
				Examples:  []string{"example1.raml"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.index.Check()
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_ReadIndex(t *testing.T) {
	validIndexContent := []byte(`{
		"package_id": "test.pkg",
		"ramlx_version": "1.0",
		"apis": ["api.raml"],
		"entities": ["entity.raml"],
		"examples": ["example.raml"]
	}`)
	
	initIndexFixture(t, validIndexContent)
	
	idx, err := ReadIndex(filepath.Join("testdata", "indexes"))
	require.NoError(t, err)
	require.NotNil(t, idx)
	require.Equal(t, "test.pkg", idx.PackageID)
}

func Test_DecodeIndex(t *testing.T) {
	validJSON := `{
		"package_id": "test.pkg",
		"ramlx_version": "1.0",
		"apis": ["api.raml"],
		"entities": ["entity.raml"],
		"examples": ["example.raml"]
	}`
	
	invalidJSON := `{
		"package_id": "test.pkg",
		"ramlx_version": "1.0",
		"apis": ["api.raml"],
		"entities": ["entity.raml"],
		"examples": ["example.raml"],
		invalid json
	}`

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{"ValidJSON", validJSON, false},
		{"InvalidJSON", invalidJSON, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := DecodeIndex(strings.NewReader(tt.content))
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, idx)
				require.Equal(t, "test.pkg", idx.PackageID)
			}
		})
	}
}

func Test_GenerateIndexRaml(t *testing.T) {
	idx := &Index{
		Entities: []string{"entity1.raml", "entity2.raml"},
		Examples: []string{"example1.raml", "example2.raml"},
	}

	tests := []struct {
		name           string
		includeExamples bool
		expected       string
	}{
		{
			name:           "WithoutExamples",
			includeExamples: false,
			expected:       "#%RAML 1.0 Library\nuses:\n  e1: entity1.raml\n  e2: entity2.raml",
		},
		{
			name:           "WithExamples",
			includeExamples: true,
			expected:       "#%RAML 1.0 Library\nuses:\n  e1: entity1.raml\n  e2: entity2.raml\n  x1: example1.raml\n  x2: example2.raml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idx.GenerateIndexRaml(tt.includeExamples)
			require.Equal(t, tt.expected, result)
		})
	}
}

func Test_Clone(t *testing.T) {
	original := &Index{
		PackageID: "test.pkg",
		Apis:      []string{"api1.raml"},
		Entities:  []string{"entity1.raml"},
		Examples:  []string{"example1.raml"},
	}

	cloned := original.Clone()
	require.Equal(t, original, cloned)
	
	// Modify cloned to ensure deep copy
	cloned.PackageID = "modified.pkg"
	require.NotEqual(t, original.PackageID, cloned.PackageID)
}

func Test_ToBytes(t *testing.T) {
	idx := &Index{
		PackageID: "test.pkg",
		Apis:      []string{"api1.raml"},
		Entities:  []string{"entity1.raml"},
		Examples:  []string{"example1.raml"},
	}

	bytes := idx.ToBytes()
	require.NotEmpty(t, bytes)

	// Verify it can be decoded back
	var decoded Index
	err := json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)
	require.Equal(t, idx, &decoded)
}

func Test_Save(t *testing.T) {
	idx := &Index{
		PackageID: "test.pkg",
		Apis:      []string{"api1.raml"},
		Entities:  []string{"entity1.raml"},
		Examples:  []string{"example1.raml"},
	}

	testPath := filepath.Join("testdata", "indexes")
	require.NoError(t, os.RemoveAll(testPath))
	require.NoError(t, os.MkdirAll(testPath, os.ModePerm))

	err := idx.Save(testPath)
	require.NoError(t, err)

	// Verify file was created and can be read back
	savedIdx, err := ReadIndexFile(filepath.Join(testPath, "index.json"))
	require.NoError(t, err)
	require.Equal(t, idx, savedIdx)
}

func Test_PutSerialized(t *testing.T) {
	idx := &Index{
		Serialized: []string{"file1.json"},
	}

	// Test adding new file
	idx.PutSerialized("file2.json")
	require.Equal(t, []string{"file1.json", "file2.json"}, idx.Serialized)

	// Test adding duplicate file
	idx.PutSerialized("file1.json")
	require.Equal(t, []string{"file1.json", "file2.json"}, idx.Serialized)
}

func Test_GetEntities(t *testing.T) {
	idx := &Index{
		Entities: []string{"entity1.raml", "entity2.raml"},
	}

	entities, err := idx.GetEntities()
	require.NoError(t, err)
	require.Len(t, entities, 2)
	require.Equal(t, "entity1", entities[0].Name)
	require.Equal(t, "entity1.raml", entities[0].Path)
	require.Equal(t, "entity2", entities[1].Name)
	require.Equal(t, "entity2.raml", entities[1].Path)
}

func Test_GetAssets(t *testing.T) {
	idx := &Index{
		Assets: []string{"asset1.png", "asset2.jpg"},
	}

	assets := idx.GetAssets()
	require.Equal(t, []string{"asset1.png", "asset2.jpg"}, assets)
}
