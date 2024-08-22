/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package identifier

import (
	"crypto/md5"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		{
			name:       "error, minor is absent",
			input:      "cti.a.p.gr.namespace.v777",
			wantErrMsg: "parse entity name and version: minor part of version is missing",
		},
		{
			name:       "error, version is absent",
			input:      "cti.a.p.gr.namespace.v",
			wantErrMsg: "parse entity name and version: version is missing",
		},
		{
			name:       "error, minor version is wildcard",
			input:      "cti.a.p.gr.namespace.v777.*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, version is wildcard",
			input:      "cti.a.p.gr.namespace.v*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, entity name ends with wildcard",
			input:      "cti.a.p.gr.namespace.*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, entity name is wildcard",
			input:      "cti.a.p.*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, package is wildcard",
			input:      "cti.a.*",
			wantErrMsg: "parse package: wildcard is disabled",
		},
		{
			name:       "error, vendor is wildcard",
			input:      "cti.*",
			wantErrMsg: "parse vendor: wildcard is disabled",
		},
		{
			name:  "ok, normal version",
			input: "cti.a.p.gr.namespace.v77.11",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{Major: NullVersion{77, true}, Minor: NullVersion{Value: 11, Valid: true}},
			}},
		},
		{
			name:       "error, invalid version, 0.0",
			input:      "cti.a.p.gr.namespace.v0.0",
			wantErrMsg: `parse entity name and version: version must be higher than 0.0`,
		},
		{
			name:       "error, invalid version, 0",
			input:      "cti.a.p.gr.namespace.v0",
			wantErrMsg: `parse entity name and version: minor part of version is missing`,
		},
		{
			name:       "error, query is disabled",
			input:      `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantErrMsg: `expect "~", got "["`,
		},
		{
			name:       "error, attribute is disabled",
			input:      `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantErrMsg: `expect "~", got "@"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExp, gotErr := ParseIdentifier(tt.input)
			if tt.wantErrMsg != "" {
				require.EqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			require.NoError(t, gotErr)
			gotExp.parser = nil
			require.EqualValues(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			require.Equal(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func TestParseAttribute(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		{
			name:       "error, version is absent",
			input:      "cti.a.p.gr.namespace.v@meta.status.name_1",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		{
			name:       "error, version is wildcard",
			input:      "cti.a.p.gr.namespace.v*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		{
			name:       "error, minor version is wildcard",
			input:      "cti.a.p.gr.namespace.v1.*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		{
			name:       "error, entity name ends with wildcard",
			input:      "cti.a.p.gr.namespace.*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		{
			name:       "error, entity name is wildcard",
			input:      "cti.a.p.*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		{
			name:       "error, package is wildcard",
			input:      "cti.a.*@meta.status.name_1",
			wantErrMsg: "parse package: wildcard is disabled",
		},
		{
			name:       "error, vendor is wildcard",
			input:      "cti.*@meta.status.name_1",
			wantErrMsg: "parse vendor: wildcard is disabled",
		},
		{
			name:       "error, query is disabled",
			input:      `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantErrMsg: `expect "~", got "["`,
		},
		{
			name:  "ok, attribute is enabled",
			input: `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
				Child: &Node{
					Vendor:     Vendor("a"),
					Package:    Package("p"),
					EntityName: EntityName("gr.datacenter"),
					Version:    NewVersion(2, 1),
				},
			}, AttributeSelector: "meta.status.name_1"},
		},
		{
			name:  "ok, attribute is enabled, no minor version",
			input: `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2@meta.status.name_1`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
				Child: &Node{
					Vendor:     Vendor("a"),
					Package:    Package("p"),
					EntityName: EntityName("gr.datacenter"),
					Version:    NewPartialVersion(2),
				},
			}, AttributeSelector: "meta.status.name_1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExp, gotErr := ParseAttributeSelector(tt.input)
			if tt.wantErrMsg != "" {
				require.EqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			require.NoError(t, gotErr)
			gotExp.parser = nil
			require.EqualValues(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			require.Equal(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		{
			name:  "ok, query is enabled",
			input: `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{"status", QueryAttributeValue{Raw: "active"}},
			}},
		},
		{
			name:  "ok, query is enabled, no minor version",
			input: `cti.a.p.gr.namespace.v1[status="active"]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewPartialVersion(1),
			}, QueryAttributes: []QueryAttribute{
				{"status", QueryAttributeValue{Raw: "active"}},
			}},
		},
		{
			name:       "error, version is absent",
			input:      `cti.a.p.gr.namespace.v[status="active"]`,
			wantErrMsg: "parse entity name and version: version is missing",
		},
		{
			name:       "error, version is wildcard",
			input:      `cti.a.p.gr.namespace.v*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, minor version is wildcard",
			input:      `cti.a.p.gr.namespace.v1.*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, entity name ends with wildcard",
			input:      `cti.a.p.gr.namespace.*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, entity name is wildcard",
			input:      `cti.a.p.*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		{
			name:       "error, package is wildcard",
			input:      `cti.a.*[status="active"]`,
			wantErrMsg: "parse package: wildcard is disabled",
		},
		{
			name:       "error, vendor is wildcard",
			input:      `cti.*[status="active"]`,
			wantErrMsg: "parse vendor: wildcard is disabled",
		},
		{
			name:       "error, attribute is disabled",
			input:      `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantErrMsg: `expect "~", got "@"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExp, gotErr := ParseQuery(tt.input)
			if tt.wantErrMsg != "" {
				require.EqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			require.NoError(t, gotErr)
			gotExp.parser = nil
			require.EqualValues(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			require.Equal(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func TestParseReference(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		{
			name:       "error, query is disabled",
			input:      `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantErrMsg: `expect "~", got "["`,
		},
		{
			name:       "error, attribute is disabled",
			input:      `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantErrMsg: `expect "~", got "@"`,
		},
		{
			name:       "error, no version",
			input:      `cti.a.p.gr.namespace`,
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		{
			name:  "ok, full version",
			input: `cti.a.p.gr.namespace.v1.0`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}},
		},
		{
			name:  "ok, minor version is absent",
			input: `cti.a.p.gr.namespace.v1`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewPartialVersion(1),
			}},
		},
		{
			name:  "ok, version is absent",
			input: `cti.a.p.gr.namespace.v`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{},
			}},
		},
		{
			name:  "ok, minor version is wildcard",
			input: `cti.a.p.gr.namespace.v1.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{Major: NullVersion{Value: 1, Valid: true}, HasMinorWildcard: true},
			}},
		},
		{
			name:  "ok, version is wildcard",
			input: `cti.a.p.gr.namespace.v*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{HasMajorWildcard: true},
			}},
		},
		{
			name:  "ok, entity name ends with wildcard",
			input: `cti.a.p.gr.namespace.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace.*"),
				Version:    Version{},
			}},
		},
		{
			name:  "ok, entity name is wildcard",
			input: `cti.a.p.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("*"),
				Version:    Version{},
			}},
		},
		{
			name:  "ok, package is wildcard",
			input: `cti.a.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("*"),
				EntityName: EntityName(""),
				Version:    Version{},
			}},
		},
		{
			name:  "ok, vendor is wildcard",
			input: `cti.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("*"),
				Package:    Package(""),
				EntityName: EntityName(""),
				Version:    Version{},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExp, gotErr := ParseReference(tt.input)
			if tt.wantErrMsg != "" {
				require.EqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			require.NoError(t, gotErr)
			gotExp.parser = nil
			require.EqualValues(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			require.Equal(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func TestParser_Parse(t *testing.T) {
	parser := NewParser(WithAllowAnonymousEntity(true))

	tests := []struct {
		name       string
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		{
			name:       "error, empty string",
			input:      "",
			wantErrMsg: "not CTI expression",
		},
		{
			name:       "error, not CTI expression",
			input:      "foo.bar",
			wantErrMsg: "not CTI expression",
		},
		{
			name:       "error, dangling separator",
			input:      "cti.a.p.gr.namespace.v1.2~",
			wantErrMsg: `unexpected dangling separator "~"`,
		},

		// Common tests
		{
			name:       "ok, simple CTI",
			input:      "cti.a.p.gr.namespace.v1.2",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 2),
			}},
		},
		{
			name:       "ok, simple CTI, inheritance",
			input:      "cti.a.p.gr.namespace.v1.2~a.p.integrations.datacenters.v2.1",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 2),
				Child: &Node{
					Vendor:     Vendor("a"),
					Package:    Package("p"),
					EntityName: EntityName("integrations.datacenters"),
					Version:    NewVersion(2, 1),
				},
			}},
		},
		{
			name:       "ok, simple CTI, inheritance, underscore in entity name",
			input:      "cti.a.p.dts.func.v2.1~a.p.access_policies._roles_ids_.any_of.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("dts.func"),
				Version:    NewVersion(2, 1),
				Child: &Node{
					Vendor:     Vendor("a"),
					Package:    Package("p"),
					EntityName: EntityName("access_policies._roles_ids_.any_of"),
					Version:    NewVersion(1, 0),
				},
			}},
		},
		{
			name:       "error, wildcard not at the end",
			input:      "cti.a.p.gr.namespace.v1.*~a.*",
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},

		// Tests for "vendor" part
		{
			name:       "ok, wildcard in vendor",
			input:      "cti.*",
			wantErrMsg: "",
			wantExp:    Expression{Head: &Node{Vendor: Vendor("*")}},
		},
		{
			name:       "ok, wildcard in vendor, inheritance",
			input:      "cti.a.p.gr.namespace.v1.2~a.*",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 2),
				Child: &Node{
					Vendor:  Vendor("a"),
					Package: Package("*"),
				},
			}},
		},
		{
			name:       "error, empty vendor",
			input:      "cti..foo",
			wantErrMsg: "parse vendor: cannot be empty",
		},
		{
			name:       "error, invalid vendor, wildcard in prefix",
			input:      "cti.*foo",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		{
			name:       "error, invalid vendor, wildcard in postfix",
			input:      "cti.foo*",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		{
			name:       "error, vendor contains invalid charts",
			input:      "cti.foo!bar",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},

		// Tests for "package" part
		{
			name:       "error, empty package",
			input:      "cti.a..foo",
			wantErrMsg: "parse package: cannot be empty",
		},
		{
			name:    "ok, wildcard in package",
			input:   "cti.acronis.*",
			wantExp: Expression{Head: &Node{Vendor: Vendor("acronis"), Package: Package("*")}},
		},
		{
			name:       "ok, wildcard in package, inheritance",
			input:      "cti.a.p.gr.namespace.v1.2~b.*",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 2),
				Child: &Node{
					Vendor:  Vendor("b"),
					Package: Package("*"),
				},
			}},
		},
		{
			name:       "error, invalid vendor, wildcard in prefix",
			input:      "cti.a.*foo",
			wantErrMsg: `parse package: can be "*" or contain only lower letters, digits, and "_"`,
		},
		{
			name:       "error, invalid vendor, wildcard in postfix",
			input:      "cti.a.foo*",
			wantErrMsg: `parse package: can be "*" or contain only lower letters, digits, and "_"`,
		},
		{
			name:       "error, vendor contains invalid charts",
			input:      "cti.a.foo!bar",
			wantErrMsg: `parse package: can be "*" or contain only lower letters, digits, and "_"`,
		},

		// Tests for "entity name" part
		{
			name:       "error, invalid entity name, invalid char",
			input:      "cti.a.p.name!space.v1.1",
			wantErrMsg: `parse entity name and version: entity name can be "*" or contain only lower letters, digits, "." and "_"`,
		},
		{
			name:       "error, invalid entity name, starts with digit",
			input:      "cti.a.p.1a.v1.1",
			wantErrMsg: `parse entity name and version: entity name can be "*" or start only with letter`,
		},
		{
			name:       "error, invalid entity name, starts with dot",
			input:      "cti.a.p..v1.1",
			wantErrMsg: `parse entity name and version: entity name can be "*" or start only with letter or "_"`,
		},
		{
			name:       "error, invalid entity name, double dots",
			input:      "cti.a.p.gr..namespace.v1.1",
			wantErrMsg: `parse entity name and version: entity name cannot have double dots ("..")`,
		},
		{
			name:       "error, invalid entity name, double underscores",
			input:      "cti.a.p.gr__namespace.v1.1",
			wantErrMsg: `parse entity name and version: entity name cannot have double underscores ("__")`,
		},
		{
			name:       "ok, wildcard in entity name",
			input:      "cti.a.p.*",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("*"),
			}},
		},
		{
			name:       "ok, wildcard in composite entity name, dot delimiter",
			input:      "cti.a.p.gr.*",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.*"),
			}},
		},
		{
			name:       "ok, entity name is just an underscore",
			input:      "cti.a.p._.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("_"),
				Version:    NewVersion(1, 0),
			}},
		},
		{
			name:       "ok, entity name starts with underscore",
			input:      "cti.a.p._abc.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("_abc"),
				Version:    NewVersion(1, 0),
			}},
		},
		{
			name:       "ok, entity name ends with underscore",
			input:      "cti.a.p.abc_.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("abc_"),
				Version:    NewVersion(1, 0),
			}},
		},
		{
			name:       "ok, entity name has underscore after and before dot",
			input:      "cti.a.p._a_._b_._c_.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("_a_._b_._c_"),
				Version:    NewVersion(1, 0),
			}},
		},
		{
			name:       "error, wildcard in entity name, after letter",
			input:      "cti.a.p.gr*",
			wantErrMsg: `parse entity name and version: wildcard "*" in entity name may be only after dot (".")`,
		},
		{
			name:       "error, wildcard in entity name, after underscore",
			input:      "cti.a.p.gr_*",
			wantErrMsg: `parse entity name and version: wildcard "*" in entity name may be only after dot (".")`,
		},
		{
			name:       "error, wildcard in entity name, not in the end",
			input:      "cti.a.p.*gr",
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, wildcard in composite entity name, not after dot",
			input:      "cti.a.p.gr.namespace*",
			wantErrMsg: `parse entity name and version: wildcard "*" in entity name may be only after dot (".")`,
		},
		{
			name:       "error, wildcard in composite entity name, not after dot",
			input:      "cti.a.p.gr.*namespace",
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "ok, wildcard in entity name, inheritance",
			input:      "cti.a.p.gr.namespace.v1.2~b.c.*",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 2),
				Child: &Node{
					Vendor:     Vendor("b"),
					Package:    Package("c"),
					EntityName: EntityName("*"),
				},
			}},
		},

		// Tests for "version" part
		{
			name:       "error, version is missing",
			input:      "cti.a.p.gr.namespace",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		{
			name:       "error, version is missing",
			input:      "cti.a.p.gr.namespace.v",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		{
			name:       "error, version is missing",
			input:      "cti.a.p.gr.namespace.v1",
			wantErrMsg: `parse entity name and version: minor part of version is missing`,
		},
		{
			name:       "error, version is missing, entity name has underscore before v1.0",
			input:      "cti.a.p.a_v1.0",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		{
			name:  "ok, wildcard in major version",
			input: "cti.a.p.gr.namespace.v*",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{HasMajorWildcard: true},
			}},
		},
		{
			name:  "ok, wildcard in minor version",
			input: "cti.a.p.gr.namespace.v777.*",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{Major: NullVersion{777, true}, HasMinorWildcard: true},
			}},
		},
		{
			name:       "error, invalid major version, unexpected char",
			input:      "cti.a.p.gr.namespace.v1*",
			wantErrMsg: `parse entity name and version: major part of version is invalid`,
		},
		{
			name:       "error, invalid minor version, leading zero",
			input:      "cti.a.p.gr.namespace.v1.01",
			wantErrMsg: `parse entity name and version: minor part of version cannot contain leading zero`,
		},
		{
			name:       "error, invalid version, 0.0",
			input:      "cti.a.p.gr.namespace.v0.0",
			wantErrMsg: `parse entity name and version: version must be higher than 0.0`,
		},
		{
			name:  "ok, simple CTI, version at the end",
			input: "cti.a.p.gr.namespace.v1.0",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}},
		},
		{
			name:  "ok, major version == 0 [VP-728]",
			input: "cti.a.p.gr.namespace.v0.1",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(0, 1),
			}},
		},
		{
			name:       "error, invalid version, 0.0",
			input:      "cti.a.p.gr.namespace.v0.0",
			wantErrMsg: `parse entity name and version: version must be higher than 0.0`,
		},
		{
			name:       "error, invalid version, 0 in case of minor is not optional",
			input:      "cti.a.p.gr.namespace.v0",
			wantErrMsg: `parse entity name and version: minor part of version is missing`,
		},

		// Tests for query attributes
		{
			name:  "ok, query, single attr, plain value, double quote",
			input: `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{"status", QueryAttributeValue{Raw: "active"}},
			}},
		},
		{
			name:  "ok, query, multiple attrs, plain values, spaces",
			input: `cti.a.p.gr.namespace.v1.0[   status =  "active" , name= 'tenants' ]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{"status", QueryAttributeValue{Raw: "active"}},
				{"name", QueryAttributeValue{Raw: "tenants"}},
			}},
			wantExpStr: `cti.a.p.gr.namespace.v1.0[status="active",name="tenants"]`,
		},
		{
			name:       "error, query, minor version is wildcard",
			input:      `cti.a.p.gr.namespace.v1.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, version is wildcard",
			input:      `cti.a.p.gr.namespace.v*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, entity name ends with wildcard",
			input:      `cti.a.p.gr.namespace.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, package is wildcard",
			input:      `cti.a.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, vendor is wildcard",
			input:      `cti.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, unexpected end of string",
			input:      `cti.a.p.gr.namespace.v1.0[`,
			wantErrMsg: `parse query attributes: unexpected end of string`,
		},
		{
			name:       "error, query, attr is not started with letter",
			input:      `cti.a.p.gr.namespace.v1.0[123`,
			wantErrMsg: `parse query attributes: attribute name and its each part should start with letter`,
		},
		{
			name:       "error, query, attr is not started with letter, spaces",
			input:      `cti.a.p.gr.namespace.v1.0[   123`,
			wantErrMsg: `parse query attributes: attribute name and its each part should start with letter`,
		},
		{
			name:       "error, query, = is missing after attr name",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123`,
			wantErrMsg: `parse query attributes: expect "=", got end of string`,
		},
		{
			name:       "error, query, attr value is missing",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = `,
			wantErrMsg: `parse query attributes: expect attribute value, got end of string`,
		},
		{
			name:       "error, query, unexpected end of string",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = val`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		{
			name:       "error, query, attr name is empty",
			input:      `cti.a.p.gr.namespace.v1.0[=`,
			wantErrMsg: `parse query attributes: attribute name cannot be empty and should contain only letters, digits, ".", and "_"`,
		},
		{
			name:       "error, query, closing quote is missing",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = '`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		{
			name:       "error, query, closing quote is missing",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = '"`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		{
			name:       "error, query, closing double quote is missing",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = "'`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		{
			name:       "error, query, attr value is empty",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = ''`,
			wantErrMsg: `parse query attributes: attribute value cannot be empty`,
		},
		{
			name:       "error, query, quote in attr value is not escaped",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo''`,
			wantErrMsg: `parse query attributes: expect ",", got "'"`,
		},
		{
			name:       "error, query, multiple attrs, invalid char in attr name",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo',&*]`,
			wantErrMsg: `parse query attributes: attribute name cannot be empty and should contain only letters, digits, ".", and "_"`,
		},
		{
			name:       "error, query, not at the end",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo']~a.*`,
			wantErrMsg: `expression may have query only at the end`,
		},
		{
			name:       "error, query, wrong attrs delimiter",
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo' | attr_321 = 'bar']`,
			wantErrMsg: `parse query attributes: expect ",", got "|"`,
		},
		{
			name:       "error, query, unexpected end of string",
			input:      `cti.a.p.gr.namespace.v1.0[foo`,
			wantErrMsg: `parse query attributes: expect "=", got end of string`,
		},
		{
			name:       "error, query, double dots in attr name",
			input:      `cti.a.p.gr.namespace.v1.0[meta..name=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name cannot have double dots ("..")`,
		},
		{
			name:       "error, query, attr name starts with dot",
			input:      `cti.a.p.gr.namespace.v1.0[.name=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name should start with letter`,
		},
		{
			name:       "error, query, attr name ends with dot",
			input:      `cti.a.p.gr.namespace.v1.0[meta.=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name cannot end with dot (".")`,
		},
		{
			name:       "error, query, not letter after dot in attr name",
			input:      `cti.a.p.gr.namespace.v1.0[meta.123=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name and its each part should start with letter`,
		},
		{
			name:  "ok, query, single attr, plain value, no quotes",
			input: `cti.a.p.gr.namespace.v1.0[attr_1=val_1]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{"attr_1", QueryAttributeValue{Raw: `val_1`}},
			}},
			wantExpStr: `cti.a.p.gr.namespace.v1.0[attr_1="val_1"]`,
		},
		{
			name:  "ok, query, single attr, plain value, dots in name, no quotes",
			input: `cti.a.p.gr.namespace.v1.0[meta.name=ns_name]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{"meta.name", QueryAttributeValue{Raw: `ns_name`}},
			}},
			wantExpStr: `cti.a.p.gr.namespace.v1.0[meta.name="ns_name"]`,
		},
		{
			name:  "ok, query, single attr, plain value, double quotes, escaping",
			input: `cti.a.p.gr.namespace.v1.0[attr_1="foo \\\"bar\""]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{"attr_1", QueryAttributeValue{Raw: `foo \\"bar"`}},
			}},
		},
		{
			name:  "ok, query, single attr, CTI value, double quotes, escaping",
			input: `cti.a.p.em.event.v1.0[ topic="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0" ]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("em.event"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{
					Name: "topic",
					Value: QueryAttributeValue{
						Raw: `cti.a.p.em.topic.v1.0~a.p.tenant.v1.0`,
						Expression: Expression{parser: parser, Head: &Node{
							Vendor:     Vendor("a"),
							Package:    Package("p"),
							EntityName: EntityName("em.topic"),
							Version:    NewVersion(1, 0),
							Child: &Node{
								Vendor:     Vendor("a"),
								Package:    Package("p"),
								EntityName: EntityName("tenant"),
								Version:    NewVersion(1, 0),
							},
						}},
					},
				},
			}},
			wantExpStr: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0"]`,
		},
		{
			name:  "ok, query, single attr, CTI value, no quotes, escaping",
			input: `cti.a.p.em.event.v1.0[ topic=cti.a.p.em.topic.v1.0~a.p.tenant.v1.0 ]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("em.event"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{
					Name: "topic",
					Value: QueryAttributeValue{
						Raw: `cti.a.p.em.topic.v1.0~a.p.tenant.v1.0`,
						Expression: Expression{parser: parser, Head: &Node{
							Vendor:     Vendor("a"),
							Package:    Package("p"),
							EntityName: EntityName("em.topic"),
							Version:    NewVersion(1, 0),
							Child: &Node{
								Vendor:     Vendor("a"),
								Package:    Package("p"),
								EntityName: EntityName("tenant"),
								Version:    NewVersion(1, 0),
							},
						}},
					},
				},
			}},
			wantExpStr: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0"]`,
		},
		{
			name:  "ok, query, single attr, value is anonymous entity, no quotes, escaping",
			input: `cti.a.p.em.event.v1.0[ topic=cti.a.p.em.topic.v1.0~c78aad06-6ef8-4267-a0f3-175e5f582754 ]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("em.event"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{
					Name: "topic",
					Value: QueryAttributeValue{
						Raw: `cti.a.p.em.topic.v1.0~c78aad06-6ef8-4267-a0f3-175e5f582754`,
						Expression: Expression{parser: parser, Head: &Node{
							Vendor:     Vendor("a"),
							Package:    Package("p"),
							EntityName: EntityName("em.topic"),
							Version:    NewVersion(1, 0),
						}, AnonymousEntityUUID: uuid.NullUUID{UUID: uuid.MustParse("c78aad06-6ef8-4267-a0f3-175e5f582754"), Valid: true}},
					},
				},
			}},
			wantExpStr: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~c78aad06-6ef8-4267-a0f3-175e5f582754"]`,
		},

		// Tests for attribute selector
		{
			name:  "ok, attribute selector, simple name",
			input: `cti.a.p.gr.namespace.v1.0@status`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, AttributeSelector: "status"},
		},
		{
			name:  "ok, attribute selector, composite name, inheritance",
			input: `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
				Child: &Node{
					Vendor:     Vendor("a"),
					Package:    Package("p"),
					EntityName: EntityName("gr.datacenter"),
					Version:    NewVersion(2, 1),
				},
			}, AttributeSelector: "meta.status.name_1"},
		},
		{
			name:       "error, attribute selector, name is empty",
			input:      `cti.a.p.gr.namespace.v1.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, version is wildcard",
			input:      `cti.a.p.gr.namespace.v*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, entity name ends with wildcard",
			input:      `cti.a.p.gr.namespace.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, package is wildcard",
			input:      `cti.a.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, query, vendor is wildcard",
			input:      `cti.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		{
			name:       "error, attribute selector, name is empty",
			input:      `cti.a.p.gr.namespace.v1.0@`,
			wantErrMsg: `parse attribute selector: attribute name cannot be empty and should contain only letters, digits, ".", and "_"`,
		},
		{
			name:       "error, attribute selector, name has double dots",
			input:      `cti.a.p.gr.namespace.v1.0@status..name`,
			wantErrMsg: `parse attribute selector: attribute name cannot have double dots ("..")`,
		},
		{
			name:       "error, attribute selector, name starts with digit",
			input:      `cti.a.p.gr.namespace.v1.0@42`,
			wantErrMsg: `parse attribute selector: attribute name and its each part should start with letter`,
		},

		// Tests for CTI expressions with anonymous entity
		{
			name:  "ok, anonymous entity, uuid at the end",
			input: "cti.a.p.gr.namespace.v1.2~550e8400-e29b-41d4-a716-446655440000",
			wantExp: Expression{
				Head: &Node{
					Vendor:     Vendor("a"),
					Package:    Package("p"),
					EntityName: EntityName("gr.namespace"),
					Version:    NewVersion(1, 2),
				},
				AnonymousEntityUUID: uuid.NullUUID{
					UUID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
					Valid: true,
				},
			},
		},
		{
			name:       "error, anonymous entity, only one uuid is allowed",
			input:      "cti.a.p.gr.namespace.v1.2~550e8400-e29b-41d4-a716-446655440000~25e34468-3e2c-4441-8482-db515ab2d032",
			wantErrMsg: `expression may have anonymous entity UUID only at the end`,
		},
		{
			name:       "error, anonymous entity, uuid at the end but only xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx format is allowed, so it's interpreted as vendor",
			input:      "cti.a.p.gr.namespace.v1.2~{550e8400-e29b-41d4-a716-446655440000}",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		{
			name:  "ok, anonymous entity with attribute selector",
			input: `cti.a.p.gr.namespace.v1.0~ba3c448e-55e3-4f7f-ae54-4e87eb8635f6@status`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			},
				AttributeSelector:   "status",
				AnonymousEntityUUID: uuid.NullUUID{UUID: uuid.MustParse("ba3c448e-55e3-4f7f-ae54-4e87eb8635f6"), Valid: true},
			},
		},
		{
			name:  "ok, anonymous entity with query, single attr, plain value",
			input: `cti.a.p.gr.namespace.v1.0~e64db2eb-1d7c-4d66-b610-5c214f5a0cf4[attr_1=val_1]`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, QueryAttributes: []QueryAttribute{
				{"attr_1", QueryAttributeValue{Raw: `val_1`}},
			}, AnonymousEntityUUID: uuid.NullUUID{UUID: uuid.MustParse("e64db2eb-1d7c-4d66-b610-5c214f5a0cf4"), Valid: true}},
			wantExpStr: `cti.a.p.gr.namespace.v1.0~e64db2eb-1d7c-4d66-b610-5c214f5a0cf4[attr_1="val_1"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExp, gotErr := parser.Parse(tt.input)
			if tt.wantErrMsg != "" {
				require.EqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			require.NoError(t, gotErr)
			tt.wantExp.parser = parser
			require.EqualValues(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			require.Equal(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func TestMustParse(t *testing.T) {
	require.PanicsWithError(t, "not CTI expression", func() {
		MustParse("foo.bar")
	})
}

// ---------------------- Benchmarks ----------------------

var benchParseExprIdentifiers = []string{
	"cti.a.p.gr.namespace.v1.2~a.p.integrations.datacenters.v2.1",
	"cti.a.p.gr.namespace.v1.0~a.p.web_restore_user_links.v1.0",
	"cti.a.p.stm.s3_buckets_pool.v1.0~my_vendor.my_app.assests.v1.0",
	"cti.a.p.wm.workload.v1.0~a.p.wm.aspect.v1.0~a.p.machine.v1.0",
}

var benchParseExprWildcards = []string{
	"cti.a.p.wr.report_config.v1.0~*",
	"cti.a.p.gr.namespace.v1.0~a.p.web_restore.*",
	"cti.a.p.wr.report_config.v1.0~a.p.mc.alerts_report.v1.0~a.p.mc.alerts_report.v1.*",
}

func BenchmarkParser_Parse_Identifier(b *testing.B) {
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(benchParseExprIdentifiers[i%len(benchParseExprIdentifiers)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParser_Parse_Wildcard(b *testing.B) {
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(benchParseExprWildcards[i%len(benchParseExprWildcards)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParser_Parse_Query(b *testing.B) {
	rawExp := `cti.a.p.am.alert.v1.0[type="cti.a.p.am.alert.v1.0~sophos.endpoint_protection.*"]`
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(rawExp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRegExp_Parse_Identifier(b *testing.B) {
	// RegExp source: https://git.acronis.work/projects/APPS/repos/acronis-platform/browse/common/cti.raml?until=ce2a13dcd2ea804eda6d8db0855411c871e56f6c&untilPath=common%2Fcti.raml#170
	regExp := regexp.MustCompile(`^cti\.([a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z_][a-z0-9_.]*\.v[\d]+\.[\d]+)(~([a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z_][a-z0-9_.]*\.v[\d]+\.[\d]+))*(~[0-9a-f]{8}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{12})?$`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := benchParseExprIdentifiers[i%len(benchParseExprIdentifiers)]
		if !regExp.MatchString(s) {
			b.Fatalf("%q is not matched", s)
		}
	}
}

func BenchmarkRegExp_Parse_Wildcard(b *testing.B) {
	// RegExp source: https://git.acronis.work/projects/APPS/repos/acronis-platform/browse/common/cti.raml?until=ce2a13dcd2ea804eda6d8db0855411c871e56f6c&untilPath=common%2Fcti.raml#198
	regExp := regexp.MustCompile(`^cti((\.([a-z][a-z0-9_]*))|\.)?(\.([a-z][a-z0-9_]*))?(\.([a-z_][a-z0-9_.]*))?(\.v(\d+|\d*\.\d*|\d*\.)?)?(~(([a-z][a-z0-9_]*)|([a-z][a-z0-9_]*)\.)?(\.([a-z][a-z0-9_]*))?(\.([a-z_][a-z0-9_.]*))?(\.v(\d+|\d*\.\d*|\d*\.)?)?)*\*$|^cti\.([a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z_][a-z0-9_.]*\.v[\d]+\.[\d]+)(~([a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z_][a-z0-9_.]*\.v[\d]+\.[\d]+))*(~[0-9a-f]{8}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{12})?$`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := benchParseExprWildcards[i%len(benchParseExprWildcards)]
		if !regExp.MatchString(s) {
			b.Fatalf("%q is not matched", s)
		}
	}
}

// BenchmarkMD5 is added just for comparison.
func BenchmarkMD5(b *testing.B) {
	expBytes := make([][]byte, len(benchParseExprIdentifiers))
	for i := range expBytes {
		expBytes[i] = []byte(benchParseExprIdentifiers[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = md5.Sum(expBytes[i%len(expBytes)])
	}
}
