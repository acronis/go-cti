package parser_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/pkg/parser"
)

func getIndexFilePath() string {
	wd := os.Getenv("TestDir")
	fmt.Println(wd)
	path := filepath.Join(wd, ".platform", "index.json")
	return path
}

func TestNewRamlParser(t *testing.T) {
	path := getIndexFilePath()
	ramlParser, _ := parser.NewRamlParser(path)

	require.NotEmpty(t, ramlParser)
	if ramlParser.Path != path {
		t.Errorf("Expected path to be %q, but got %q", path, ramlParser.Path)
	}
}

func TestParse(t *testing.T) {
	var parserErrors parser.ParserErrors

	path := getIndexFilePath()
	ramlParser, _ := parser.NewRamlParser(path)
	parserOutputs, err := ramlParser.ParseAll()

	// Verify Error object
	require.Nil(t, err)

	if err != nil {
		errData := []byte(err.Error())
		parserErrs := json.Unmarshal(errData, &parserErrors)
		if parserErrs == nil {
			for _, pErr := range parserErrors.Errors {
				require.Empty(t, pErr.Location)
				require.Empty(t, pErr.Message)
			}
		}
	}

	require.NotEmpty(t, parserOutputs)

	require.Nil(t, err)
	for _, output := range parserOutputs {
		require.NotEmpty(t, output.Cti)
		require.NotNil(t, output.Final)
	}
}

func TestParsePlatformIndexFile(t *testing.T) {
	var parserErrors parser.ParserErrors

	path := getIndexFilePath()
	ramlParser, _ := parser.NewRamlParser(path)
	parserOutputs, err := ramlParser.ParseAll()

	// Verify Error object
	require.Nil(t, err)

	if err != nil {
		errData := []byte(err.Error())
		parserErrs := json.Unmarshal(errData, &parserErrors)
		if parserErrs == nil {
			for _, pErr := range parserErrors.Errors {
				require.Empty(t, pErr.Location)
				require.Empty(t, pErr.Message)
			}
		}
	}

	require.NotEmpty(t, parserOutputs)

	require.Nil(t, err)
	for _, output := range parserOutputs {
		require.NotEmpty(t, output.Cti)
		require.NotNil(t, output.Final)
	}
}

func TestCtiFinal(t *testing.T) {
	var parserErrors parser.ParserErrors

	path := getIndexFilePath()
	ramlParser, _ := parser.NewRamlParser(path)
	parserOutputs, err := ramlParser.ParseAll()

	// Verify Error object
	require.Nil(t, err)

	if err != nil {
		errData := []byte(err.Error())
		parserErrs := json.Unmarshal(errData, &parserErrors)
		if parserErrs == nil {
			for _, pErr := range parserErrors.Errors {
				require.Empty(t, pErr.Location)
				require.Empty(t, pErr.Message)
			}
		}
	}

	require.NotEmpty(t, parserOutputs)

	require.Nil(t, err)

	for _, output := range parserOutputs {
		if output.Schema == nil {
			continue
		}

		schema := output.Schema
		jsonSchema := make(map[string]interface{})
		err = json.Unmarshal(schema, &jsonSchema)
		require.Nil(t, err)

		// Get schema definitions
		xDomainExtCtiFinal := "x-domainExt-cti.final"
		definitions := jsonSchema["definitions"].(map[string]interface{})
		for _, value := range definitions {
			schemaDefinition := value.(map[string]interface{})
			if schemaDefinition[xDomainExtCtiFinal] == nil {
				require.Equal(t, output.Final, true)
			} else {
				require.Equal(t, output.Final, schemaDefinition[xDomainExtCtiFinal])
			}
		}
	}
}

func TestCtiSchema(t *testing.T) {
	var parserErrors parser.ParserErrors

	path := getIndexFilePath()
	ramlParser, _ := parser.NewRamlParser(path)
	parserOutputs, err := ramlParser.ParseAll()

	// Verify Error object
	require.Nil(t, err)

	if err != nil {
		errData := []byte(err.Error())
		parserErrs := json.Unmarshal(errData, &parserErrors)
		if parserErrs == nil {
			for _, pErr := range parserErrors.Errors {
				require.Empty(t, pErr.Location)
				require.Empty(t, pErr.Message)
			}
		}
	}

	require.NotEmpty(t, parserOutputs)

	require.Nil(t, err)

	getSchemaAnnotation := func(jsonObject map[string]interface{}, cbFunc callback) {
		for _, value := range jsonObject {
			iterateJSON(value, cbFunc)
		}
	}

	for _, output := range parserOutputs {
		if output.Schema == nil {
			continue
		}

		schema := output.Schema
		jsonSchema := make(map[string]interface{})
		err = json.Unmarshal(schema, &jsonSchema)
		require.Nil(t, err)

		getSchemaAnnotation(jsonSchema, getSchemaAnnotation)
	}
}

type callback func(jsonObject map[string]interface{}, cbFunc callback)

func iterateJSON(value interface{}, cbFunc callback) {
	//nolint
	switch value.(type) {
	case map[string]interface{}:
		// Value is another JSON object
		subObject := value.(map[string]interface{})
		cbFunc(subObject, cbFunc)
	case []interface{}:
		// Value is an array
		array := value.([]interface{})
		iterateArray(array, cbFunc)
	default:
		// Item is a primitive type
		break
	}
}

func iterateArray(array []interface{}, cbFunc callback) {
	for _, item := range array {
		//nolint
		switch item.(type) {
		case map[string]interface{}:
			// Item is another JSON object
			subObject := item.(map[string]interface{})
			iterateJSON(subObject, cbFunc)
		case []interface{}:
			// Item is an array
			subArray := item.([]interface{})
			iterateArray(subArray, cbFunc)
		default:
			// Item is a primitive type
			continue
		}
	}
}

func TestParser_ParseAllWithHeapSizeLimit(t *testing.T) {
	path := getIndexFilePath()
	nonExistentIndexPath := filepath.Join(os.TempDir(), "index.json")
	tests := []struct {
		name          string
		path          string
		parserOutputs parser.CtiEntities
		err           string
	}{
		{
			name:          "returns errors when parse (huge) .platform",
			path:          path,
			parserOutputs: nil,
			err:           "signal: abort",
		},
		{
			name:          "returns error when invalid path is provided",
			path:          "invalid/path",
			parserOutputs: nil,
			err:           "no such file or directory",
		},
		{
			name:          "returns error when file does not exist",
			path:          nonExistentIndexPath,
			parserOutputs: nil,
			err:           "error parsing index file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ramlParser, _ := parser.NewRamlParser(tt.path)
			ramlParser.RamlX.SetMaxHeapSize(200)
			parserOutputs, err := ramlParser.ParseAll()

			require.Contains(t, err.Error(), tt.err)
			require.Equal(t, tt.parserOutputs, parserOutputs)
		})
	}
}

func TestValidateParserOutput_OK(t *testing.T) {
	outputs, err := parser.ValidateParserOutput(entities)
	require.Nil(t, err)
	require.NotNil(t, outputs)
	require.Equal(t, 2, len(outputs))
}

var entities = []byte(`[
	  {
		"final": true,
		"cti": "cti.a.p.dts.func.v1.0~a.p.uie.workload_actions.action.interface.v1.0~vendor1.protect.protect.v1.0",
		"schema": {
		  "$schema": "http://json-schema.org/draft-07/schema#",
		  "$ref": "#/definitions/FuncUIeWorkloadActionsActionProtect",
		  "definitions": {
			"FuncUIeWorkloadActionsActionProtect": {
			  "x-domainExt-cti.cti": "cti.a.p.dts.func.v1.0~a.p.uie.workload_actions.action.interface.v1.0~vendor1.protect.protect.v1.0",
			  "x-shapeExt-data-cti-traits": {
				"defaults": {
				  "dictionaries": [
					"cti.a.p.presentation.dict.v1.0~vendor1.protect.dictionary.v1.0"
				  ]
				},
				"return": {
				  "id": "cti.a.p.dts.ui.item.v1.0~a.p.uie.workload_actions.action.v1.0~vendor1.protect.protect.v1.0",
				  "meta": {
					"order": 1,
					"ext": {
					  "icon": "protect_icon",
					  "label": "$trans(\"Protect\")",
					  "action_container_type": ""
					}
				  }
				},
				"context": {
				  "workload": "$func[\u003cget_workload\u003e](workload_ids=.input.workload_ids[0])",
				  "current_tenant": "$func[cti.a.p.dts.func.v1.0~a.p.get_tenant.v1.0]()"
				},
				"entrypoint": true,
				"deterministic_behavior": "QUERY"
			  },
			  "description": "The 'Workload Actions' 'Protect' action constructor",
			  "type": "object",
			  "additionalProperties": false
			}
		  }
		},
		"traits": {
		  "defaults": {
			"dictionaries": [
			  "cti.a.p.presentation.dict.v1.0~vendor1.protect.dictionary.v1.0"
			]
		  },
		  "return": {
			"id": "cti.a.p.dts.ui.item.v1.0~a.p.uie.workload_actions.action.v1.0~vendor1.protect.protect.v1.0",
			"meta": {
			  "order": 1,
			  "ext": {
				"icon": "protect_icon",
				"label": "$trans(\"Protect\")",
				"action_container_type": ""
			  }
			}
		  },
		  "context": {
			"workload": "$func[\u003cget_workload\u003e](workload_ids=.input.workload_ids[0])",
			"current_tenant": "$func[cti.a.p.dts.func.v1.0~a.p.get_tenant.v1.0]()"
		  },
		  "entrypoint": true,
		  "deterministic_behavior": "QUERY"
		},
		"annotations": {
		  ".": "cti.a.p.dts.func.v1.0~a.p.uie.workload_actions.action.interface.v1.0~vendor1.protect.protect.v1.0",
		  "$name": "UIeWorkloadActionsActionInterface",
		  "$sourcePath": ".platform/dts/ui/uie_workload_actions/types/interfaces.raml",
		  "$originalPath": "entities/protect-action/1-vendor/constructors.raml"
		}
	  },
	  {
		"final": true,
		"cti": "cti.a.p.dts.processing_step.v1.0~a.productization.uie.workload_actions.protect.v1.0",
		"values": {
		  "subjects": [
			"cti.a.p.dts.func.v1.0~a.p.uie.workload_actions.action.interface.v1.0~vendor1.protect.protect.v1.0"
		  ],
		  "description": "The 'Workload Actions' 'Protect' action processing step",
		  "context": {
			"me": "$func[cti.a.p.dts.func.v1.0~a.p.get_user.v1.0]()"
		  },
		  "filter": "$.context.me.is_admin",
		  "id": "cti.a.p.dts.processing_step.v1.0~a.productization.uie.workload_actions.protect.v1.0",
		  "deterministic_behavior": "QUERY"
		},
		"annotations": {
		  "$annotationType": {
			"name": "DTSProcessingSteps",
			"type": "array",
			"reference": ".platform/dts/types.raml"
		  },
		  "$originalPath": "entities/protect-action/2-acronis/processing_steps.raml",
		  "$sourcePath": "entities/protect-action/2-acronis/processing_steps.raml"
		}
	  }
	]`)

var InvalidEntitiesOutput = []byte(`{"final": true, "annotations": {}}`)

func TestValidateParserOutput_Err(t *testing.T) {
	_, err := parser.ValidateParserOutput(InvalidEntitiesOutput)
	require.NotNil(t, err)
}

var InvalidEntitiesOutput2 = []byte(`{"final": true, "cti": "", schema": {}, "annotations": {}}`)

func TestValidateParserOutput_Err2(t *testing.T) {
	_, err := parser.ValidateParserOutput(InvalidEntitiesOutput2)
	require.NotNil(t, err)
}
