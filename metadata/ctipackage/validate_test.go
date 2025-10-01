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
	packagePath := `` // put path to the package to test here

	if packagePath == "" {
		t.Skip("packagePath is empty, skipping manual test")
	}

	// Create and parse the package
	pkg, err := New(packagePath)
	require.NoError(t, err)
	require.NoError(t, pkg.Read())
	require.NoError(t, pkg.Validate())
}

func Test_CTISchemaUsage(t *testing.T) {
	testsupp.InitLog(t)

	testCases := map[string]struct {
		testsupp.PackageTestCase
		wantError bool
		validate  func(*Package)
	}{
		"cti type based on cti schema": {
			PackageTestCase: testsupp.PackageTestCase{
				PkgId:    "x.y",
				Entities: []string{"types.raml"},
				Files: map[string]string{"types.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  Schema1:
    (cti.cti): cti.x.y.schema_1.v1.0
    (cti.final): false
    properties:
      schema_prop1:
        type: object

  Schema2:
    (cti.cti): cti.x.y.schema_2.v1.0
    (cti.final): false
    properties:
      schema_prop2:
        type: object

  Schema3:
    (cti.cti): cti.x.y.schema_3.v1.0
    (cti.final): false
    (cti.schema):
    - cti.x.y.schema_1.v1.0
    - cti.x.y.schema_2.v1.0
`)},
			},
			validate: func(pkg *Package) {
				golden := `{
  "cti.x.y.schema_1.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/Schema1",
    "definitions": {
      "Schema1": {
        "properties": {
          "schema_prop1": {
            "type": "object"
          }
        },
        "type": "object",
        "required": [
          "schema_prop1"
        ],
        "x-cti.cti": "cti.x.y.schema_1.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.schema_2.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/Schema2",
    "definitions": {
      "Schema2": {
        "properties": {
          "schema_prop2": {
            "type": "object"
          }
        },
        "type": "object",
        "required": [
          "schema_prop2"
        ],
        "x-cti.cti": "cti.x.y.schema_2.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.schema_3.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/Schema3",
    "definitions": {
      "Schema3": {
        "anyOf": [
          {
            "properties": {
              "schema_prop1": {
                "type": "object"
              }
            },
            "type": "object",
            "required": [
              "schema_prop1"
            ],
            "x-cti.cti": "cti.x.y.schema_1.v1.0",
            "x-cti.final": false
          },
          {
            "properties": {
              "schema_prop2": {
                "type": "object"
              }
            },
            "type": "object",
            "required": [
              "schema_prop2"
            ],
            "x-cti.cti": "cti.x.y.schema_2.v1.0",
            "x-cti.final": false
          }
        ],
        "x-cti.schema": [
          "cti.x.y.schema_1.v1.0",
          "cti.x.y.schema_2.v1.0"
        ]
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
		"invalid self recursive cti.schema": {
			PackageTestCase: testsupp.PackageTestCase{
				PkgId:    "x.y",
				Entities: []string{"types.raml"},
				Files: map[string]string{"types.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  Schema1:
    (cti.cti): cti.x.y.schema_1.v1.0
    (cti.final): false
    properties:
      schema_prop2:
        type: object

  InvalidSelfRecursive:
    (cti.cti): cti.x.y.schema_2.v1.0
    (cti.final): false
    (cti.schema): cti.x.y.schema_2.v1.0

  InvalidSelfRecursiveUnion:
    (cti.cti): cti.x.y.schema_3.v1.0
    (cti.final): false
    (cti.schema):
    - cti.x.y.schema_1.v1.0
    - cti.x.y.schema_3.v1.0
`)},
			},
			validate:  func(pkg *Package) {},
			wantError: true,
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
			err = pkg.Parse()
			if tc.wantError {
				require.Error(t, err, "Expected parsing error for package %s", name)
				slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
				return
			}
			require.NoError(t, err)

			{
				err = pkg.Validate()
				if tc.wantError {
					require.Error(t, err, "Expected error for package %s", name)
					slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
				} else {
					if err != nil {
						slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
						t.Fatalf("Unexpected error for package %s: %v", name, err)
					}
				}
				tc.validate(pkg)
			}
		})
	}
}

func Test_ImplicitCTISchema(t *testing.T) {
	testsupp.InitLog(t)

	testCases := map[string]struct {
		testsupp.PackageTestCase
		wantError bool
		validate  func(*Package)
	}{
		"implicit cti schema": {
			PackageTestCase: testsupp.PackageTestCase{
				PkgId:    "x.y",
				Entities: []string{"types.raml"},
				Files: map[string]string{"types.raml": strings.TrimSpace(`
#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  Schema:
    (cti.cti): cti.x.y.schema.v1.0
    (cti.final): false
    properties:
      schema_prop:
        type: object

  ImplicitEmbed:
    (cti.cti): cti.x.y.implicit_embed.v1.0
    (cti.final): false
    properties:
      prop:
        type: Schema

  ImplicitEmbedAlias:
    (cti.cti): cti.x.y.implicit_embed_alias.v1.0
    (cti.final): false
    properties:
      prop: Schema

  ImplicitEmbedUnion:
    (cti.cti): cti.x.y.implicit_embed_union.v1.0
    (cti.final): false
    properties:
      prop:
        type: Schema | nil

  ImplicitEmbedAliasedUnion:
    (cti.cti): cti.x.y.implicit_embed_aliased_union.v1.0
    (cti.final): false
    properties:
      prop: Schema | nil

  ImplicitEmbedInvalidUnion:
    (cti.cti): cti.x.y.implicit_embed_invalid_union.v1.0
    (cti.final): false
    properties:
      prop: Schema | string | nil
`)},
			},
			validate: func(pkg *Package) {
				golden := `{
  "cti.x.y.implicit_embed.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/ImplicitEmbed",
    "definitions": {
      "ImplicitEmbed": {
        "properties": {
          "prop": {
            "properties": {
              "schema_prop": {
                "type": "object"
              }
            },
            "type": "object",
            "required": [
              "schema_prop"
            ],
            "x-cti.cti": "cti.x.y.schema.v1.0",
            "x-cti.final": false,
            "x-cti.schema": "cti.x.y.schema.v1.0"
          }
        },
        "type": "object",
        "required": [
          "prop"
        ],
        "x-cti.cti": "cti.x.y.implicit_embed.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.implicit_embed_alias.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/ImplicitEmbedAlias",
    "definitions": {
      "ImplicitEmbedAlias": {
        "properties": {
          "prop": {
            "properties": {
              "schema_prop": {
                "type": "object"
              }
            },
            "type": "object",
            "required": [
              "schema_prop"
            ],
            "x-cti.cti": "cti.x.y.schema.v1.0",
            "x-cti.final": false,
            "x-cti.schema": "cti.x.y.schema.v1.0"
          }
        },
        "type": "object",
        "required": [
          "prop"
        ],
        "x-cti.cti": "cti.x.y.implicit_embed_alias.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.implicit_embed_aliased_union.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/ImplicitEmbedAliasedUnion",
    "definitions": {
      "ImplicitEmbedAliasedUnion": {
        "properties": {
          "prop": {
            "anyOf": [
              {
                "properties": {
                  "schema_prop": {
                    "type": "object"
                  }
                },
                "type": "object",
                "required": [
                  "schema_prop"
                ],
                "x-cti.cti": "cti.x.y.schema.v1.0",
                "x-cti.final": false
              },
              {
                "type": "null"
              }
            ],
            "x-cti.schema": [
              "cti.x.y.schema.v1.0",
              null
            ]
          }
        },
        "type": "object",
        "required": [
          "prop"
        ],
        "x-cti.cti": "cti.x.y.implicit_embed_aliased_union.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.implicit_embed_invalid_union.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/ImplicitEmbedInvalidUnion",
    "definitions": {
      "ImplicitEmbedInvalidUnion": {
        "properties": {
          "prop": {
            "anyOf": [
              {
                "properties": {
                  "schema_prop": {
                    "type": "object"
                  }
                },
                "type": "object",
                "required": [
                  "schema_prop"
                ],
                "x-cti.cti": "cti.x.y.schema.v1.0",
                "x-cti.final": false,
                "x-cti.schema": "cti.x.y.schema.v1.0"
              },
              {
                "type": "string"
              },
              {
                "type": "null"
              }
            ]
          }
        },
        "type": "object",
        "required": [
          "prop"
        ],
        "x-cti.cti": "cti.x.y.implicit_embed_invalid_union.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.implicit_embed_union.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/ImplicitEmbedUnion",
    "definitions": {
      "ImplicitEmbedUnion": {
        "properties": {
          "prop": {
            "anyOf": [
              {
                "properties": {
                  "schema_prop": {
                    "type": "object"
                  }
                },
                "type": "object",
                "required": [
                  "schema_prop"
                ],
                "x-cti.cti": "cti.x.y.schema.v1.0",
                "x-cti.final": false
              },
              {
                "type": "null"
              }
            ],
            "x-cti.schema": [
              "cti.x.y.schema.v1.0",
              null
            ]
          }
        },
        "type": "object",
        "required": [
          "prop"
        ],
        "x-cti.cti": "cti.x.y.implicit_embed_union.v1.0",
        "x-cti.final": false
      }
    }
  },
  "cti.x.y.schema.v1.0": {
    "$schema": "http://json-schema.org/draft-07/schema",
    "$ref": "#/definitions/Schema",
    "definitions": {
      "Schema": {
        "properties": {
          "schema_prop": {
            "type": "object"
          }
        },
        "type": "object",
        "required": [
          "schema_prop"
        ],
        "x-cti.cti": "cti.x.y.schema.v1.0",
        "x-cti.final": false
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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			pkg, err := New(testsupp.InitTestPackageFiles(t, name, tc.PackageTestCase),
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
					require.Error(t, err, "Expected error for package %s", name)
					slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
				} else {
					if err != nil {
						slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
						t.Fatalf("Unexpected error for package %s: %v", name, err)
					}
				}
				tc.validate(pkg)
			}
		})
	}
}

func Test_RecursiveSchemas(t *testing.T) {
	testsupp.InitLog(t)

	testCases := map[string]struct {
		testsupp.PackageTestCase
		wantError bool
		validate  func(*Package)
	}{
		"recursive types validation": {
			PackageTestCase: testsupp.PackageTestCase{
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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			pkg, err := New(testsupp.InitTestPackageFiles(t, name, tc.PackageTestCase),
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
					require.Error(t, err, "Expected error for package %s", name)
					slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
				} else {
					if err != nil {
						slog.Error("Command failed", slogex.ErrToSlogAttr(err, stacktrace.WithEnsureDuplicates()))
						t.Fatalf("Unexpected error for package %s: %v", name, err)
					}
				}
				tc.validate(pkg)
			}
		})
	}
}
