package ctipackage

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-cti/metadata/testsupp"
	"github.com/acronis/go-stacktrace"
	slogex "github.com/acronis/go-stacktrace/slogex"
	"github.com/stretchr/testify/require"
)

func TestValidateManual(t *testing.T) {
	testsupp.ManualTest(t, "manual test for package validation")

	packagePath := ``

	// Create and parse the package
	pkg, err := New(packagePath)
	if err != nil {
		t.Fatalf("Failed to create package: %v", err)
	}
	if err = pkg.Read(); err != nil {
		t.Fatalf("Failed to read package: %v", err)
	}
	if err = pkg.Validate(); err != nil {
		t.Fatalf("Failed to validate package: %v", err)
	}
}

func Test_PackageValidations(t *testing.T) {
	testsupp.InitLog(t)

	type testCase struct {
		testsupp.PackageTestCase
		wantError bool
		validate  func(*Package)
	}

	testCases := []testCase{
		{
			PackageTestCase: testsupp.PackageTestCase{
				Name:     "recursive types validation",
				PkgId:    "x.y",
				Entities: []string{"types.raml"},
				Files: map[string]string{"types.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  PlainParent:
    (cti.cti): cti.x.y.plain_parent.v1.0
    (cti.final): false
    properties:
      external_recursion:
        type: object

  RecursiveChild:
    type: PlainParent
    (cti.cti): cti.x.y.plain_parent.v1.0~x.y.recursive_child.v1.0
    properties:
      external_recursion:
        type: object
        (cti.schema): cti.x.y.self_recursive_parent.v1.0

  CrossRecursiveParent:
    (cti.cti): cti.x.y.cross_recursive_parent.v1.0
    additionalProperties: false
    properties:
      self_recursion?:
        type: object
        (cti.schema): cti.x.y.cross_recursive_parent.v1.0
      cross_self_recursion?:
        type: object
        (cti.schema): cti.x.y.self_recursive_parent.v1.0

  SelfRecursiveParent:
    (cti.cti): cti.x.y.self_recursive_parent.v1.0
    properties:
      cross_recursion:
        type: object
        (cti.schema): cti.x.y.cross_recursive_parent.v1.0
      self_recursion:
        type: object
        (cti.schema): cti.x.y.self_recursive_parent.v1.0
`)},
			},
			validate: func(pkg *Package) {
				golden := `{
  "cti.x.y.cross_recursive_parent.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/CrossRecursiveParent",
    "definitions": {
      "CrossRecursiveParent": {
        "properties": {
          "self_recursion": {
            "$ref": "#",
            "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
          },
          "cross_self_recursion": {
            "properties": {
              "cross_recursion": {
                "$ref": "#",
                "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
              },
              "self_recursion": {
                "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
              }
            },
            "type": "object",
            "required": [
              "cross_recursion",
              "self_recursion"
            ],
            "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
          }
        },
        "additionalProperties": false,
        "type": "object",
        "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0"
      },
      "cti.x.y.self_recursive_parent.v1.0": {
        "properties": {
          "cross_recursion": {
            "properties": {
              "self_recursion": {
                "$ref": "#",
                "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
              },
              "cross_self_recursion": {
                "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
              }
            },
            "additionalProperties": false,
            "type": "object",
            "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
          },
          "self_recursion": {
            "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
            "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
          }
        },
        "type": "object",
        "required": [
          "cross_recursion",
          "self_recursion"
        ],
        "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
        "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
      }
    }
  },
  "cti.x.y.plain_parent.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/PlainParent",
    "definitions": {
      "PlainParent": {
        "properties": {
          "external_recursion": {
            "type": "object"
          }
        },
        "type": "object",
        "required": [
          "external_recursion"
        ],
        "x-cti.cti": "cti.x.y.plain_parent.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.plain_parent.v1.0~x.y.recursive_child.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/RecursiveChild",
    "definitions": {
      "RecursiveChild": {
        "properties": {
          "external_recursion": {
            "properties": {
              "cross_recursion": {
                "properties": {
                  "self_recursion": {
                    "$ref": "#/definitions/cti.x.y.cross_recursive_parent.v1.0",
                    "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                    "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
                  },
                  "cross_self_recursion": {
                    "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
                    "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                    "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
                  }
                },
                "additionalProperties": false,
                "type": "object",
                "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
              },
              "self_recursion": {
                "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
              }
            },
            "type": "object",
            "required": [
              "cross_recursion",
              "self_recursion"
            ],
            "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
          }
        },
        "type": "object",
        "required": [
          "external_recursion"
        ],
        "x-cti.cti": "cti.x.y.plain_parent.v1.0~x.y.recursive_child.v1.0"
      },
      "cti.x.y.cross_recursive_parent.v1.0": {
        "properties": {
          "self_recursion": {
            "$ref": "#/definitions/cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
          },
          "cross_self_recursion": {
            "properties": {
              "cross_recursion": {
                "$ref": "#/definitions/cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
              },
              "self_recursion": {
                "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
              }
            },
            "type": "object",
            "required": [
              "cross_recursion",
              "self_recursion"
            ],
            "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
          }
        },
        "additionalProperties": false,
        "type": "object",
        "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
        "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
      },
      "cti.x.y.self_recursive_parent.v1.0": {
        "properties": {
          "cross_recursion": {
            "properties": {
              "self_recursion": {
                "$ref": "#/definitions/cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
              },
              "cross_self_recursion": {
                "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
              }
            },
            "additionalProperties": false,
            "type": "object",
            "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
          },
          "self_recursion": {
            "$ref": "#/definitions/cti.x.y.self_recursive_parent.v1.0",
            "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
          }
        },
        "type": "object",
        "required": [
          "cross_recursion",
          "self_recursion"
        ],
        "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
        "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
      }
    }
  },
  "cti.x.y.self_recursive_parent.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/SelfRecursiveParent",
    "definitions": {
      "SelfRecursiveParent": {
        "properties": {
          "cross_recursion": {
            "properties": {
              "self_recursion": {
                "$ref": "#/definitions/cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
              },
              "cross_self_recursion": {
                "$ref": "#",
                "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
              }
            },
            "additionalProperties": false,
            "type": "object",
            "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
          },
          "self_recursion": {
            "$ref": "#",
            "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
          }
        },
        "type": "object",
        "required": [
          "cross_recursion",
          "self_recursion"
        ],
        "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0"
      },
      "cti.x.y.cross_recursive_parent.v1.0": {
        "properties": {
          "self_recursion": {
            "$ref": "#/definitions/cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
          },
          "cross_self_recursion": {
            "properties": {
              "cross_recursion": {
                "$ref": "#/definitions/cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
              },
              "self_recursion": {
                "$ref": "#",
                "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
                "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
              }
            },
            "type": "object",
            "required": [
              "cross_recursion",
              "self_recursion"
            ],
            "x-cti.cti": "cti.x.y.self_recursive_parent.v1.0",
            "x-cti.schema": "cti.x.y.self_recursive_parent.v1.0"
          }
        },
        "additionalProperties": false,
        "type": "object",
        "x-cti.cti": "cti.x.y.cross_recursive_parent.v1.0",
        "x-cti.schema": "cti.x.y.cross_recursive_parent.v1.0"
      }
    }
  }
}`
				mergedSchemas := make(map[string]*jsonschema.JSONSchemaCTI)
				for _, et := range pkg.GlobalRegistry.Types {
					s, err := et.GetMergedSchema()
					require.NoError(t, err, "Failed to get merged schema for %s", et.CTI)
					mergedSchemas[et.CTI] = s
				}
				b, err := json.MarshalIndent(mergedSchemas, "", "  ")
				require.NoError(t, err, "Failed to marshal merged schemas")
				require.Equal(t, golden, string(b), "Merged schemas do not match")
			},
			wantError: false,
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
			require.NoError(t, pkg.Parse())

			{
				err = pkg.Validate()
				if tc.wantError {
					require.Error(t, err, "Expected error for package %s", tc.Name)
					slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
				} else {
					if err != nil {
						slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
						t.Fatalf("Unexpected error for package %s: %v", tc.Name, err)
					}
				}
				tc.validate(pkg)
			}
		})
	}
}
