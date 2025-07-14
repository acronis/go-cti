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
		"depends": {"dep": "1.0.0"},
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
		"depends": {"dep": "1.0.0"},
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

func Test_DecodeIndex(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name: "ValidIndex",
			input: `{"package_id":"test.pkg","apis":["api.raml"]}
`,
			expectError: false,
		},
		{
			name:        "InvalidJSON",
			input:       `{invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			idx, err := DecodeIndex(reader)
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

func Test_Clone(t *testing.T) {
	original := &Index{
		PackageID:    "test.pkg",
		RamlxVersion: "1.0",
		Apis:         []string{"api.raml"},
		Entities:     []string{"entity.raml"},
		Depends:      map[string]string{"dep": "1.0"},
	}

	cloned := original.Clone()

	require.NotSame(t, original, cloned)
	require.Equal(t, original.PackageID, cloned.PackageID)
	require.Equal(t, original.RamlxVersion, cloned.RamlxVersion)
	require.Equal(t, original.Apis, cloned.Apis)
	require.Equal(t, original.Entities, cloned.Entities)
	require.Equal(t, original.Depends, cloned.Depends)
}

func Test_ToBytes(t *testing.T) {
	idx := &Index{
		PackageID: "test.pkg",
		Apis:      []string{"api.raml"},
	}

	bytes := idx.ToBytes()

	// Verify the bytes can be unmarshaled back to the same structure
	var decoded Index
	require.NoError(t, json.Unmarshal(bytes, &decoded))
	require.Equal(t, idx.PackageID, decoded.PackageID)
	require.Equal(t, idx.Apis, decoded.Apis)
}

func Test_Save(t *testing.T) {
	testDir := filepath.Join("testdata", "save_test")
	require.NoError(t, os.RemoveAll(testDir))
	require.NoError(t, os.MkdirAll(testDir, os.ModePerm))

	idx := &Index{
		PackageID: "test.pkg",
		Apis:      []string{"api.raml"},
	}

	require.NoError(t, idx.Save(testDir))

	// Verify the file was created and contains correct content
	content, err := os.ReadFile(filepath.Join(testDir, IndexFileName))
	require.NoError(t, err)

	var decoded Index
	require.NoError(t, json.Unmarshal(content, &decoded))
	require.Equal(t, idx.PackageID, decoded.PackageID)
	require.Equal(t, idx.Apis, decoded.Apis)
}

func Test_PutSerialized(t *testing.T) {
	tests := []struct {
		name           string
		initSerialized []string
		newFile        string
		expected       []string
	}{
		{
			name:           "AddNewFile",
			initSerialized: []string{"file1.json"},
			newFile:        "file2.json",
			expected:       []string{"file1.json", "file2.json"},
		},
		{
			name:           "DuplicateFile",
			initSerialized: []string{"file1.json"},
			newFile:        "file1.json",
			expected:       []string{"file1.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := &Index{Serialized: tt.initSerialized}
			idx.PutSerialized(tt.newFile)
			require.Equal(t, tt.expected, idx.Serialized)
		})
	}
}

func Test_GetEntities(t *testing.T) {
	idx := &Index{
		Entities: []string{"path/to/entity1.raml", "path/to/entity2.raml"},
	}

	entities, err := idx.GetEntities()
	require.NoError(t, err)
	require.Len(t, entities, 2)
	require.Equal(t, "entity1", entities[0].Name)
	require.Equal(t, "path/to/entity1.raml", entities[0].Path)
	require.Equal(t, "entity2", entities[1].Name)
	require.Equal(t, "path/to/entity2.raml", entities[1].Path)
}

func Test_GetAssets(t *testing.T) {
	assets := []string{"asset1.png", "asset2.jpg"}
	idx := &Index{Assets: assets}

	require.Equal(t, assets, idx.GetAssets())
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
