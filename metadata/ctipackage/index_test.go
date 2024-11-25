package ctipackage

import (
	"os"
	"path/filepath"
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
