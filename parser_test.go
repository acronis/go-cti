/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package cti

import (
	"testing"

	"github.com/google/uuid"
)

func Test_ParseIdentifier(t *testing.T) {
	tests := map[string]struct {
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		"error, minor is absent": {
			input:      "cti.a.p.gr.namespace.v777",
			wantErrMsg: "parse entity name and version: minor part of version is missing",
		},
		"error, version is absent": {
			input:      "cti.a.p.gr.namespace.v",
			wantErrMsg: "parse entity name and version: version is missing",
		},
		"error, minor version is wildcard": {
			input:      "cti.a.p.gr.namespace.v777.*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, version is wildcard": {
			input:      "cti.a.p.gr.namespace.v*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, entity name ends with wildcard": {
			input:      "cti.a.p.gr.namespace.*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, entity name is wildcard": {
			input:      "cti.a.p.*",
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, package is wildcard": {
			input:      "cti.a.*",
			wantErrMsg: "parse package: wildcard is disabled",
		},
		"error, vendor is wildcard": {
			input:      "cti.*",
			wantErrMsg: "parse vendor: wildcard is disabled",
		},
		"ok, normal version": {
			input: "cti.a.p.gr.namespace.v77.11",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{Major: NullVersion{77, true}, Minor: NullVersion{Value: 11, Valid: true}},
			}},
		},
		"error, invalid version, 0.0": {
			input:      "cti.a.p.gr.namespace.v0.0",
			wantErrMsg: `parse entity name and version: version must be higher than 0.0`,
		},
		"error, invalid version, 0": {
			input:      "cti.a.p.gr.namespace.v0",
			wantErrMsg: `parse entity name and version: minor part of version is missing`,
		},
		"error, query is disabled": {
			input:      `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantErrMsg: `expect "~", got "["`,
		},
		"error, attribute is disabled": {
			input:      `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantErrMsg: `expect "~", got "@"`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotExp, gotErr := ParseIdentifier(tt.input)
			if tt.wantErrMsg != "" {
				assertEqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			assertNoError(t, gotErr)
			gotExp.parser = nil
			assertEqual(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			assertEqual(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func Test_ParseAttribute(t *testing.T) {
	tests := map[string]struct {
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		"error, version is absent": {
			input:      "cti.a.p.gr.namespace.v@meta.status.name_1",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		"error, version is wildcard": {
			input:      "cti.a.p.gr.namespace.v*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		"error, minor version is wildcard": {
			input:      "cti.a.p.gr.namespace.v1.*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		"error, entity name ends with wildcard": {
			input:      "cti.a.p.gr.namespace.*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		"error, entity name is wildcard": {
			input:      "cti.a.p.*@meta.status.name_1",
			wantErrMsg: `parse entity name and version: wildcard is disabled`,
		},
		"error, package is wildcard": {
			input:      "cti.a.*@meta.status.name_1",
			wantErrMsg: "parse package: wildcard is disabled",
		},
		"error, vendor is wildcard": {
			input:      "cti.*@meta.status.name_1",
			wantErrMsg: "parse vendor: wildcard is disabled",
		},
		"error, query is disabled": {
			input:      `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantErrMsg: `expect "~", got "["`,
		},
		"ok, attribute is enabled": {
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
		"ok, attribute is enabled, no minor version": {
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

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotExp, gotErr := ParseAttributeSelector(tt.input)
			if tt.wantErrMsg != "" {
				assertEqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			assertNoError(t, gotErr)
			gotExp.parser = nil
			assertEqual(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			assertEqual(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func Test_ParseQuery(t *testing.T) {
	tests := map[string]struct {
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		"ok, query is enabled": {
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
		"ok, query is enabled, no minor version": {
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
		"error, version is absent": {
			input:      `cti.a.p.gr.namespace.v`,
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		"error, version is wildcard": {
			input:      `cti.a.p.gr.namespace.v*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, minor version is wildcard": {
			input:      `cti.a.p.gr.namespace.v1.*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, entity name ends with wildcard": {
			input:      `cti.a.p.gr.namespace.*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, entity name is wildcard": {
			input:      `cti.a.p.*[status="active"]`,
			wantErrMsg: "parse entity name and version: wildcard is disabled",
		},
		"error, package is wildcard": {
			input:      `cti.a.*[status="active"]`,
			wantErrMsg: "parse package: wildcard is disabled",
		},
		"error, vendor is wildcard": {
			input:      `cti.*[status="active"]`,
			wantErrMsg: "parse vendor: wildcard is disabled",
		},
		"error, attribute is disabled": {
			input:      `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantErrMsg: `expect "~", got "@"`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotExp, gotErr := ParseQuery(tt.input)
			if tt.wantErrMsg != "" {
				assertEqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			assertNoError(t, gotErr)
			gotExp.parser = nil
			assertEqual(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			assertEqual(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func Test_ParseReference(t *testing.T) {
	tests := map[string]struct {
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		"error, query is disabled": {
			input:      `cti.a.p.gr.namespace.v1.0[status="active"]`,
			wantErrMsg: `expect "~", got "["`,
		},
		"error, attribute is disabled": {
			input:      `cti.a.p.gr.namespace.v1.0~a.p.gr.datacenter.v2.1@meta.status.name_1`,
			wantErrMsg: `expect "~", got "@"`,
		},
		"error, no version": {
			input:      `cti.a.p.gr.namespace`,
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		"ok, full version": {
			input: `cti.a.p.gr.namespace.v1.0`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}},
		},
		"ok, minor version is absent": {
			input: `cti.a.p.gr.namespace.v1`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewPartialVersion(1),
			}},
		},
		"ok, version is absent": {
			input: `cti.a.p.gr.namespace.v`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{},
			}},
		},
		"ok, minor version is wildcard": {
			input: `cti.a.p.gr.namespace.v1.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{Major: NullVersion{Value: 1, Valid: true}, HasMinorWildcard: true},
			}},
		},
		"ok, version is wildcard": {
			input: `cti.a.p.gr.namespace.v*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{HasMajorWildcard: true},
			}},
		},
		"ok, entity name ends with wildcard": {
			input: `cti.a.p.gr.namespace.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace.*"),
				Version:    Version{},
			}},
		},
		"ok, entity name is wildcard": {
			input: `cti.a.p.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("*"),
				Version:    Version{},
			}},
		},
		"ok, package is wildcard": {
			input: `cti.a.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("*"),
				EntityName: EntityName(""),
				Version:    Version{},
			}},
		},
		"ok, vendor is wildcard": {
			input: `cti.*`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("*"),
				Package:    Package(""),
				EntityName: EntityName(""),
				Version:    Version{},
			}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotExp, gotErr := ParseReference(tt.input)
			if tt.wantErrMsg != "" {
				assertEqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			assertNoError(t, gotErr)
			gotExp.parser = nil
			assertEqual(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			assertEqual(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func Test_Parse(t *testing.T) {
	parser := NewParser(WithAllowAnonymousEntity(true))

	tests := map[string]struct {
		input      string
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		"error, empty string": {
			input:      "",
			wantErrMsg: "not CTI expression",
		},
		"error, not CTI expression": {
			input:      "foo.bar",
			wantErrMsg: "not CTI expression",
		},
		"error, dangling separator": {
			input:      "cti.a.p.gr.namespace.v1.2~",
			wantErrMsg: `unexpected dangling separator "~"`,
		},
		// Common tests
		"ok, simple CTI": {
			input:      "cti.a.p.gr.namespace.v1.2",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 2),
			}},
		},
		"ok, simple CTI, inheritance": {
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
		"ok, simple CTI, inheritance, underscore in entity name": {
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
		"error, wildcard not at the end": {
			input:      "cti.a.p.gr.namespace.v1.*~a.*",
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		// Tests for "vendor" part
		"ok, wildcard in vendor": {
			input:      "cti.*",
			wantErrMsg: "",
			wantExp:    Expression{Head: &Node{Vendor: Vendor("*")}},
		},
		"ok, wildcard in vendor, inheritance": {
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
		"error, empty vendor": {
			input:      "cti..foo",
			wantErrMsg: "parse vendor: cannot be empty",
		},
		"error, invalid vendor, wildcard in prefix": {
			input:      "cti.*foo",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		"error, invalid vendor, wildcard in postfix": {
			input:      "cti.foo*",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		"error, vendor contains invalid charts": {
			input:      "cti.foo!bar",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		// Tests for "package" part
		"error, empty package": {
			input:      "cti.a..foo",
			wantErrMsg: "parse package: cannot be empty",
		},
		"ok, wildcard in package": {
			input:   "cti.acronis.*",
			wantExp: Expression{Head: &Node{Vendor: Vendor("acronis"), Package: Package("*")}},
		},
		"ok, wildcard in package, inheritance": {
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
		"error, invalid package, wildcard in prefix": {
			input:      "cti.a.*foo",
			wantErrMsg: `parse package: can be "*" or contain only lower letters, digits, and "_"`,
		},
		"error, invalid package, wildcard in postfix": {
			input:      "cti.a.foo*",
			wantErrMsg: `parse package: can be "*" or contain only lower letters, digits, and "_"`,
		},
		"error, package contains invalid charts": {
			input:      "cti.a.foo!bar",
			wantErrMsg: `parse package: can be "*" or contain only lower letters, digits, and "_"`,
		},
		// Tests for "entity name" part
		"error, invalid entity name, invalid char": {
			input:      "cti.a.p.name!space.v1.1",
			wantErrMsg: `parse entity name and version: entity name can be "*" or contain only lower letters, digits, "." and "_"`,
		},
		"error, invalid entity name, starts with digit": {
			input:      "cti.a.p.1a.v1.1",
			wantErrMsg: `parse entity name and version: entity name can be "*" or start only with letter`,
		},
		"error, invalid entity name, starts with dot": {
			input:      "cti.a.p..v1.1",
			wantErrMsg: `parse entity name and version: entity name can be "*" or start only with letter or "_"`,
		},
		"error, invalid entity name, double dots": {
			input:      "cti.a.p.gr..namespace.v1.1",
			wantErrMsg: `parse entity name and version: entity name cannot have double dots ("..")`,
		},
		"error, invalid entity name, double underscores": {
			input:      "cti.a.p.gr__namespace.v1.1",
			wantErrMsg: `parse entity name and version: entity name cannot have double underscores ("__")`,
		},
		"ok, wildcard in entity name": {
			input:      "cti.a.p.*",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("*"),
			}},
		},
		"ok, wildcard in composite entity name, dot delimiter": {
			input:      "cti.a.p.gr.*",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.*"),
			}},
		},
		"ok, entity name is just an underscore": {
			input:      "cti.a.p._.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("_"),
				Version:    NewVersion(1, 0),
			}},
		},
		"ok, entity name starts with underscore": {
			input:      "cti.a.p._abc.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("_abc"),
				Version:    NewVersion(1, 0),
			}},
		},
		"ok, entity name ends with underscore": {
			input:      "cti.a.p.abc_.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("abc_"),
				Version:    NewVersion(1, 0),
			}},
		},
		"ok, entity name has underscore after and before dot": {
			input:      "cti.a.p._a_._b_._c_.v1.0",
			wantErrMsg: "",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("_a_._b_._c_"),
				Version:    NewVersion(1, 0),
			}},
		},
		"error, wildcard in entity name, after letter": {
			input:      "cti.a.p.gr*",
			wantErrMsg: `parse entity name and version: wildcard "*" in entity name may be only after dot (".")`,
		},
		"error, wildcard in entity name, after underscore": {
			input:      "cti.a.p.gr_*",
			wantErrMsg: `parse entity name and version: wildcard "*" in entity name may be only after dot (".")`,
		},
		"error, wildcard in entity name, not in the end": {
			input:      "cti.a.p.*gr",
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, wildcard in composite entity name, not after dot": {
			input:      "cti.a.p.gr.namespace*",
			wantErrMsg: `parse entity name and version: wildcard "*" in entity name may be only after dot (".")`,
		},
		"error, wildcard prefix in composite entity name, not after dot": {
			input:      "cti.a.p.gr.*namespace",
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"ok, wildcard in entity name, inheritance": {
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
		"error, version is missing": {
			input:      "cti.a.p.gr.namespace",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		"error, version is missing 2": {
			input:      "cti.a.p.gr.namespace.v",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		"error, version is missing 3": {
			input:      "cti.a.p.gr.namespace.v1",
			wantErrMsg: `parse entity name and version: minor part of version is missing`,
		},
		"error, version is missing, entity name has underscore before v1.0": {
			input:      "cti.a.p.a_v1.0",
			wantErrMsg: `parse entity name and version: version is missing`,
		},
		"ok, wildcard in major version": {
			input: "cti.a.p.gr.namespace.v*",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{HasMajorWildcard: true},
			}},
		},
		"ok, wildcard in minor version": {
			input: "cti.a.p.gr.namespace.v777.*",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    Version{Major: NullVersion{777, true}, HasMinorWildcard: true},
			}},
		},
		"error, invalid major version, unexpected char": {
			input:      "cti.a.p.gr.namespace.v1*",
			wantErrMsg: `parse entity name and version: major part of version is invalid`,
		},
		"error, invalid minor version, leading zero": {
			input:      "cti.a.p.gr.namespace.v1.01",
			wantErrMsg: `parse entity name and version: minor part of version cannot contain leading zero`,
		},
		"error, invalid version, 0.0": {
			input:      "cti.a.p.gr.namespace.v0.0",
			wantErrMsg: `parse entity name and version: version must be higher than 0.0`,
		},
		"ok, simple CTI, version at the end": {
			input: "cti.a.p.gr.namespace.v1.0",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}},
		},
		"ok, major version == 0 [VP-728]": {
			input: "cti.a.p.gr.namespace.v0.1",
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(0, 1),
			}},
		},
		"error, invalid version, 0 in case of minor is not optional": {
			input:      "cti.a.p.gr.namespace.v0",
			wantErrMsg: `parse entity name and version: minor part of version is missing`,
		},

		// Tests for query attributes
		"ok, query, single attr, plain value, double quote": {
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
		"ok, query, multiple attrs, plain values, spaces": {
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
		"error, query, minor version is wildcard": {
			input:      `cti.a.p.gr.namespace.v1.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, version is wildcard": {
			input:      `cti.a.p.gr.namespace.v*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, entity name ends with wildcard": {
			input:      `cti.a.p.gr.namespace.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, package is wildcard": {
			input:      `cti.a.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, vendor is wildcard": {
			input:      `cti.*[status="active"]`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, unexpected end of string": {
			input:      `cti.a.p.gr.namespace.v1.0[`,
			wantErrMsg: `parse query attributes: unexpected end of string`,
		},
		"error, query, attr is not started with letter": {
			input:      `cti.a.p.gr.namespace.v1.0[123`,
			wantErrMsg: `parse query attributes: attribute name and its each part should start with letter`,
		},
		"error, query, attr is not started with letter, spaces": {
			input:      `cti.a.p.gr.namespace.v1.0[   123`,
			wantErrMsg: `parse query attributes: attribute name and its each part should start with letter`,
		},
		"error, query, = is missing after attr name": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123`,
			wantErrMsg: `parse query attributes: expect "=", got end of string`,
		},
		"error, query, attr value is missing": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = `,
			wantErrMsg: `parse query attributes: expect attribute value, got end of string`,
		},
		"error, query, unexpected end of string 2": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = val`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		"error, query, attr name is empty": {
			input:      `cti.a.p.gr.namespace.v1.0[=`,
			wantErrMsg: `parse query attributes: attribute name cannot be empty and should contain only letters, digits, ".", and "_"`,
		},
		"error, query, closing quote is missing": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = '`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		"error, query, closing quote is missing 2": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = '"`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		"error, query, closing double quote is missing": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = "'`,
			wantErrMsg: `parse query attributes: unexpected end of string while parsing attribute value`,
		},
		"error, query, attr value is empty": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = ''`,
			wantErrMsg: `parse query attributes: attribute value cannot be empty`,
		},
		"error, query, quote in attr value is not escaped": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo''`,
			wantErrMsg: `parse query attributes: expect ",", got "'"`,
		},
		"error, query, multiple attrs, invalid char in attr name": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo',&*]`,
			wantErrMsg: `parse query attributes: attribute name cannot be empty and should contain only letters, digits, ".", and "_"`,
		},
		"error, query, not at the end": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo']~a.*`,
			wantErrMsg: `expression may have query only at the end`,
		},
		"error, query, wrong attrs delimiter": {
			input:      `cti.a.p.gr.namespace.v1.0[attr_123 = 'foo' | attr_321 = 'bar']`,
			wantErrMsg: `parse query attributes: expect ",", got "|"`,
		},
		"error, query, unexpected end of string 3": {
			input:      `cti.a.p.gr.namespace.v1.0[foo`,
			wantErrMsg: `parse query attributes: expect "=", got end of string`,
		},
		"error, query, double dots in attr name": {
			input:      `cti.a.p.gr.namespace.v1.0[meta..name=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name cannot have double dots ("..")`,
		},
		"error, query, attr name starts with dot": {
			input:      `cti.a.p.gr.namespace.v1.0[.name=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name should start with letter`,
		},
		"error, query, attr name ends with dot": {
			input:      `cti.a.p.gr.namespace.v1.0[meta.=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name cannot end with dot (".")`,
		},
		"error, query, not letter after dot in attr name": {
			input:      `cti.a.p.gr.namespace.v1.0[meta.123=ns_name]`,
			wantErrMsg: `parse query attributes: attribute name and its each part should start with letter`,
		},
		"ok, query, single attr, plain value, no quotes": {
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
		"ok, query, single attr, plain value, dots in name, no quotes": {
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
		"ok, query, single attr, plain value, double quotes, escaping": {
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
		"ok, query, single attr, CTI value, double quotes, escaping": {
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
		"ok, query, single attr, CTI value, no quotes, escaping": {
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
		// Tests for attribute selector
		"ok, attribute selector, simple name": {
			input: `cti.a.p.gr.namespace.v1.0@status`,
			wantExp: Expression{Head: &Node{
				Vendor:     Vendor("a"),
				Package:    Package("p"),
				EntityName: EntityName("gr.namespace"),
				Version:    NewVersion(1, 0),
			}, AttributeSelector: "status"},
		},
		"ok, attribute selector, composite name, inheritance": {
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
		"error, attribute selector, name is empty": {
			input:      `cti.a.p.gr.namespace.v1.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, version is wildcard 2": {
			input:      `cti.a.p.gr.namespace.v*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, entity name starts with wildcard": {
			input:      `cti.a.p.gr.namespace.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, package starts with wildcard": {
			input:      `cti.a.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, query, vendor starts with wildcard": {
			input:      `cti.*@status`,
			wantErrMsg: `expression may have wildcard "*" only at the end`,
		},
		"error, attribute selector, is empty": {
			input:      `cti.a.p.gr.namespace.v1.0@`,
			wantErrMsg: `parse attribute selector: attribute name cannot be empty and should contain only letters, digits, ".", and "_"`,
		},
		"error, attribute selector, name has double dots": {
			input:      `cti.a.p.gr.namespace.v1.0@status..name`,
			wantErrMsg: `parse attribute selector: attribute name cannot have double dots ("..")`,
		},
		"error, attribute selector, name starts with digit": {
			input:      `cti.a.p.gr.namespace.v1.0@42`,
			wantErrMsg: `parse attribute selector: attribute name and its each part should start with letter`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotExp, gotErr := parser.Parse(tt.input)
			if tt.wantErrMsg != "" {
				assertEqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			assertNoError(t, gotErr)
			tt.wantExp.parser = parser
			assertEqual(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			assertEqual(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func Test_Parse_AnonymousEntity(t *testing.T) {
	// Tests for CTI expressions with anonymous entity
	for name, tt := range map[string]struct {
		input      string
		noSupport  bool
		wantExp    Expression
		wantExpStr string
		wantErrMsg string
	}{
		"ok, anonymous entity": {
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
		"error, parser no anonymous entities support": {
			input:      "cti.a.p.iam.role.v1.0~04100a2a-14e7-4d4e-8e7a-bdeaba3917d4",
			noSupport:  true,
			wantErrMsg: "parse vendor: can be \"*\" or start only with letter",
		},
		"error, anonymous entity, only one uuid is allowed": {
			input:      "cti.a.p.gr.namespace.v1.2~550e8400-e29b-41d4-a716-446655440000~25e34468-3e2c-4441-8482-db515ab2d032",
			wantErrMsg: `expression may have anonymous entity UUID only at the end`,
		},
		"error, anonymous entity, uuid at the end but only xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx format is allowed, so it's interpreted as vendor": {
			input:      "cti.a.p.gr.namespace.v1.2~{550e8400-e29b-41d4-a716-446655440000}",
			wantErrMsg: `parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		"ok, anonymous entity with attribute selector": {
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
		"ok, query, single attr, value is anonymous entity, no quotes, escaping": {
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
						Expression: Expression{Head: &Node{
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
		"error, query, single attr, value is anonymous entity": {
			input:      `cti.a.p.em.event.v1.0[ topic=cti.a.p.em.topic.v1.0~c78aad06-6ef8-4267-a0f3-175e5f582754 ]`,
			noSupport:  true,
			wantErrMsg: `parse query attributes: parse attribute "topic" as CTI: parse vendor: can be "*" or contain only lower letters, digits, and "_"`,
		},
		"ok, query, single attr, plain value": {
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
	} {
		t.Run(name, func(t *testing.T) {
			parser := func() *Parser {
				if tt.noSupport {
					return NewParser()
				}
				return NewParser(WithAllowAnonymousEntity(true))
			}()

			gotExp, gotErr := parser.Parse(tt.input)
			if tt.wantErrMsg != "" {
				assertEqualError(t, gotErr, tt.wantErrMsg)
				return
			}
			assertNoError(t, gotErr)
			// fixup parser value in expressions
			tt.wantExp.parser = parser
			for i := range tt.wantExp.QueryAttributes {
				if tt.wantExp.QueryAttributes[i].Value.IsExpression() {
					tt.wantExp.QueryAttributes[i].Value.Expression.parser = parser
				}
			}
			assertEqual(t, tt.wantExp, gotExp)

			wantExpStr := tt.input
			if tt.wantExpStr != "" {
				wantExpStr = tt.wantExpStr
			}
			assertEqual(t, wantExpStr, tt.wantExp.String())
		})
	}
}

func Test_MustParse(t *testing.T) {
	assertPanicsWithError(t, "not CTI expression", func() {
		_ = MustParse("")
	})
}
