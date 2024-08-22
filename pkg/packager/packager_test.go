package packager

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockRamlPackagerInterface struct {
	entities           []byte
	dictionaries       []byte
	assets             []os.File
	sourcePlatformPath string
}

func (m *mockRamlPackagerInterface) Archive() (*bytes.Buffer, error) {
	packager := NewRamlPackager(m.entities, m.dictionaries, m.assets, m.sourcePlatformPath, nil)
	return packager.Archive()
}

func TestPackagerImpl_Archive_Err(t *testing.T) {
	mockPackager := mockRamlPackagerInterface{
		entities:           nil,
		dictionaries:       nil,
		assets:             nil,
		sourcePlatformPath: "",
	}

	_, err := mockPackager.Archive()
	require.Equal(t, err, AddingPlatformPackageToArchiveError)
}

func TestPackagerImpl_Archive_Err2(t *testing.T) {
	sourcePlatformPath, err := getPlatformIndexFilePath()
	require.Equal(t, nil, err)

	mockPackager := mockRamlPackagerInterface{
		entities:           nil,
		dictionaries:       nil,
		assets:             nil,
		sourcePlatformPath: sourcePlatformPath,
	}

	vendorPackage, err := mockPackager.Archive()
	require.Equal(t, nil, err)

	expected := reflect.TypeOf(&bytes.Buffer{})
	actual := reflect.TypeOf(vendorPackage)
	require.Equal(t, expected, actual)
}

func TestPackagerImpl_Archive_Err3(t *testing.T) {
	sourcePlatformPath, err := getPlatformIndexFilePath()
	require.Equal(t, nil, err)

	mockPackager := mockRamlPackagerInterface{
		entities: []byte(`{"dts": "#%RAML 1.0 Library
		uses:
		  cti: ../.platform/common/cti.raml
		  ui_items: ../.platform/dts/ui/uie_workloads_grid/types/ui_items.raml
		types:
		  AcronisWorkloadsGridColumnAccount:
			(cti.cti): cti.a.p.dts.func.v1.0~a.p.uie.workloads_grid.column.interface.v1.0~a.p.account.v1.0
			(cti.final): true
			additionalProperties: false
			description: Adds the "Account" column to Workloads Grid
			type: ui_items.UIeWorkloadsGridColumnUIItem
		  AcronisWorkloadsGridColumnName:
			(cti.cti): cti.a.p.dts.func.v1.0~a.p.uie.workloads_grid.column.interface.v1.0~a.p.name.v1.0
			(cti.final): true
			additionalProperties: false
			description: Adds the "Name" column to Workloads Grid
			type: ui_items.UIeWorkloadsGridColumnUIItem"}
		`),
		dictionaries:       nil,
		assets:             nil,
		sourcePlatformPath: sourcePlatformPath,
	}

	_, err = mockPackager.Archive()
	require.Equal(t, err, PreparingRamlEntitiesError)
}

func TestPackagerImpl_Archive_Err4(t *testing.T) {
	sourcePlatformPath, err := getPlatformIndexFilePath()
	require.Equal(t, nil, err)
	mockRamlString := []byte(`{"entities/callbacks_x_y.raml":["uses:\n  acgw: ../.platform/acgw/types.raml\n  cti: ../.platform/common/cti.raml\n","types:\n  XConfigurationRead.OK:\n    (cti.cti): cti.a.p.acgw.response.v1.0~x.y.configuration.read.ok.v1.0\n    additionalProperties: false\n    cti-traits:\n      httpCode: 200\n    description: TODO\n    properties:\n      payload:\n        properties:\n          a:\n            type: string\n          b:\n            type: number\n          c:\n            properties:\n              d:\n                type: string\n            type: object\n        type: object\n    type: acgw.Response\n  XConfigurationWrite:\n    (cti.cti): cti.a.p.acgw.request.v1.0~x.y.configuration.write.v1.0\n    additionalProperties: false\n    description: TODO\n    properties:\n      payload:\n        properties:\n          a:\n            type: string\n          b:\n            type: number\n          c:\n            properties:\n              d:\n                type: string\n            type: object\n        type: object\n    type: acgw.Request\n  XPasswordReset:\n    (cti.cti): cti.a.p.acgw.request.v1.0~x.y.password.reset.v1.0\n    additionalProperties: false\n    description: TODO\n    properties:\n      payload:\n        properties:\n          password_id:\n            type: string\n        type: object\n    type: acgw.Request\n  XPasswordReset.NotFound:\n    (cti.cti): cti.a.p.acgw.response.v1.0~x.y.password.reset.not_found.v1.0\n    additionalProperties: false\n    cti-traits:\n      httpCode: 404\n    description: Indicates that password with requested id was not found\n    type: acgw.Response\n  XPasswordReset.OK:\n    (cti.cti): cti.a.p.acgw.response.v1.0~x.y.password.reset.ok.v1.0\n    additionalProperties: false\n    cti-traits:\n      httpCode: 200\n    description: Indicates that password successfully reset\n    type: acgw.Response\n","(acgw.Callbacks):\n- description: TODO\n  id: cti.a.p.acgw.callback.v1.0~x.y.configuration.read.v1.0\n  request: cti.a.p.acgw.request.v1.0~a.p.empty.v1.0\n  responses:\n  - cti.a.p.acgw.response.v1.0~x.y.configuration.read.ok.v1.0\n- description: TODO\n  id: cti.a.p.acgw.callback.v1.0~x.y.configuration.write.v1.0\n  request: cti.a.p.acgw.request.v1.0~x.y.configuration.write.v1.0\n  responses:\n  - cti.a.p.acgw.response.v1.0~a.p.success_no_content.v1.0\n- description: TODO\n  id: cti.a.p.acgw.callback.v1.0~x.y.password.reset.v1.0\n  request: cti.a.p.acgw.request.v1.0~x.y.password.reset.v1.0\n  responses:\n  - cti.a.p.acgw.response.v1.0~x.y.password.reset.ok.v1.0\n  - cti.a.p.acgw.response.v1.0~x.y.password.reset.not_found.v1.0\n"],"entities/endpoints.raml":["uses:\n  acgw: ../.platform/acgw/types.raml\n  cti: ../.platform/common/cti.raml\n","(acgw.Endpoints):\n- connection_info: cti.a.p.sm.setting.v1.0~a.p.api_callback_handler.v1.0~x.y.extra.v1.0\n  handlers:\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.read.v1.0\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.write.v1.0\n  id: cti.a.p.acgw.endpoint.v1.0~x.y.example.v1.0\n  url: handler.example.com\n- handlers:\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.read.v1.0\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.write.v1.0\n  id: cti.a.p.acgw.endpoint.v1.0~x.y.example_2.v1.0\n  url: handler.example_2.com\n- handlers:\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.read.v1.0\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.write.v1.0\n  id: cti.a.p.acgw.endpoint.v1.0~x.y.example_3.v1.0\n  url: handler.example_3.com\n- handlers:\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.read.v1.0\n  - id: cti.a.p.acgw.callback.v1.0~a.p.integration.write.v1.0\n  id: cti.a.p.acgw.endpoint.v1.0~x.y.example_4.v1.0\n  url: handler.example_4.com\n"]}`)
	mockPackager := mockRamlPackagerInterface{
		entities:           mockRamlString,
		dictionaries:       nil,
		assets:             nil,
		sourcePlatformPath: sourcePlatformPath,
	}

	vendorPackage, err := mockPackager.Archive()
	require.Equal(t, nil, err)

	expected := reflect.TypeOf(&bytes.Buffer{})
	actual := reflect.TypeOf(vendorPackage)
	require.Equal(t, expected, actual)
}

func getPlatformIndexFilePath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	path := filepath.Join(wd, "../..", "test")
	fmt.Println(path)
	return path, err
}
