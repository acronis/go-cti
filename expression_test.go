/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package cti

import (
	"testing"
)

func Test_Expression_InterpolateDynamicParameterValues(t *testing.T) {
	tests := map[string]struct {
		input                 string
		dynamicValues         DynamicParameterValues
		wantHasDynamicParams  bool
		wantExpression        string
		wantInterpolateErrMsg string
	}{
		"ok, no dynamic params": {
			input:          "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantExpression: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
		},
		"ok, dynamic params as CTI chunk": {
			input: "cti.${rootType}",
			dynamicValues: DynamicParameterValues{
				"rootType": "a.p.gr.namespace.v1.0",
			},
			wantHasDynamicParams: true,
			wantExpression:       "cti.a.p.gr.namespace.v1.0",
		},
		"ok, dynamic params as CTI chunk, inheritance": {
			input: "cti.a.p.gr.namespace.v1.0~${urlPathParameters[kv_namespace]}",
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[kv_namespace]": "a.p.integrations.datacenters.v1.0",
			},
			wantHasDynamicParams: true,
			wantExpression:       "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
		},
		"ok, dynamic params as full CTI": {
			input: "cti.a.p.gr.namespace.v1.0~${urlPathParameters[kv_namespace]}~a.p.integrations.cyberdc.v1.1",
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[kv_namespace]": "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			},
			wantHasDynamicParams: true,
			wantExpression:       "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0~a.p.integrations.cyberdc.v1.1", //nolint:lll
		},
		"ok, dynamic params as full CTI, prefix with wildcard": {
			input: "cti.a.p.gr.namespace.v1.0~${urlPathParameters[kv_namespace]}~a.p.integrations.cyberdc.v1.1",
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[kv_namespace]": "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			},
			wantHasDynamicParams: true,
			wantExpression:       "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0~a.p.integrations.cyberdc.v1.1", //nolint:lll
		},
		"ok, dynamic params as full CTI, prefix with wildcard, mismatch": {
			input: "cti.a.p.gr.namespace.v1.1~${urlPathParameters[kv_namespace]}",
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[kv_namespace]": "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			},
			wantHasDynamicParams:  true,
			wantInterpolateErrMsg: `"cti.a.p.gr.namespace.v1.1" and value "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0" of dynamic parameter "urlPathParameters[kv_namespace]" are not matched`, //nolint:lll
		},
		"error, dynamic parameter values do not have key": {
			input: "cti.${unsetVal}",
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[kv_namespace]": "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			},
			wantHasDynamicParams:  true,
			wantInterpolateErrMsg: `dynamic parameter values do not have "unsetVal" key`,
		},
		"error, dynamic params value is incorrect CTI chunk": {
			input: "cti.a.p.gr.namespace.v1.1~${urlPathParameters[kv_namespace]}",
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[kv_namespace]": "foo%^&*bar",
			},
			wantHasDynamicParams:  true,
			wantInterpolateErrMsg: `parse value "foo%^&*bar" of dynamic parameter "urlPathParameters[kv_namespace]" as CTI: parse vendor: can be "*" or contain only lower letters, digits, and "_"`, //nolint:lll
		},
		"error, dynamic params value is incorrect CTI": {
			input: "cti.a.p.gr.namespace.v1.1~${urlPathParameters[kv_namespace]}",
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[kv_namespace]": "cti.foo%^&*bar",
			},
			wantHasDynamicParams:  true,
			wantInterpolateErrMsg: `parse value "cti.foo%^&*bar" of dynamic parameter "urlPathParameters[kv_namespace]" as CTI: parse vendor: can be "*" or contain only lower letters, digits, and "_"`, //nolint:lll
		},
		"ok, query, dynamic param": {
			input: `cti.a.p.em.event.v1.0[topic="cti.${rootType}"]`,
			dynamicValues: DynamicParameterValues{
				"rootType": `cti.a.p.em.topic.v1.0~a.p.tenant.v1.0`,
			},
			wantHasDynamicParams: true,
			wantExpression:       `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0"]`,
		},
		"ok, query, dynamic param, inheritance": {
			input: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~${urlPathParameters[topic_name]}"]`,
			dynamicValues: DynamicParameterValues{
				"urlPathParameters[topic_name]": `cti.a.p.em.topic.v1.0~a.p.tenant.v1.0`,
			},
			wantHasDynamicParams: true,
			wantExpression:       `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0"]`,
		},
	}
	p := NewParser(WithAllowedDynamicParameterNames(
		"rootType",
		"urlPathParameters[kv_namespace]",
		"urlPathParameters[topic_name]",
		"unsetVal",
	))
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exp, parseErr := p.Parse(tt.input)
			assertNoError(t, parseErr)

			assertEqual(t, tt.wantHasDynamicParams, exp.HasDynamicParameters())

			interpolatedExp, interpolateErr := exp.InterpolateDynamicParameterValues(tt.dynamicValues)
			if tt.wantInterpolateErrMsg != "" {
				assertErrorContains(t, interpolateErr, tt.wantInterpolateErrMsg)
				return
			}
			assertNoError(t, interpolateErr)

			assertEqual(t, tt.wantExpression, interpolatedExp.String())
		})
	}
}

func Test_Expression_Match(t *testing.T) {
	tests := map[string]struct {
		expression1           string
		expression2           string
		wantMatchErrMsg       string
		ignoreQuery           bool
		expr1ParseAsReference bool
		wantMatch             bool
	}{
		"matched, exact match": {
			expression1: "cti.a.p.gr.namespace.v1.0",
			expression2: "cti.a.p.gr.namespace.v1.0",
			wantMatch:   true,
		},
		"matched, inheritance": {
			expression1: "cti.a.p.gr.namespace.v1.0",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"matched, wildcard in vendor": {
			expression1: "cti.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"matched, wildcard in vendor, inheritance": {
			expression1: "cti.a.p.gr.namespace.v1.0~*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"not matched, different vendor": {
			expression1: "cti.b.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   false,
		},
		"matched, wildcard in package": {
			expression1: "cti.a.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"matched, wildcard in package, inheritance": {
			expression1: "cti.a.p.gr.namespace.v1.0~a.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"not matched, different package": {
			expression1: "cti.a.b.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   false,
		},
		"matched, wildcard in type name": {
			expression1: "cti.a.p.gr.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"matched, wildcard in type name, inheritance": {
			expression1: "cti.a.p.gr.namespace.v1.0~a.p.integrations.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"matched, wildcard in type name, inheritance, version is right after in the 2nd expression": {
			expression1: "cti.a.p.gr.namespace.v1.0~a.p.integrations.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.v1.0",
			wantMatch:   true,
		},
		"not matched, wildcard in type name, same prefix right before wildcard": {
			expression1: "cti.a.p.gr.namespace.v1.0~a.p.data.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.datacenters.v1.0",
			wantMatch:   false,
		},
		"not matched, different type name": {
			expression1: "cti.a.p.gr.namespace.v1.0~a.p.integrations.regions.v1.0",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   false,
		},
		"matched, wildcard in major version": {
			expression1: "cti.a.p.gr.namespace.v*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"matched, wildcard in minor version": {
			expression1: "cti.a.p.gr.namespace.v1.*",
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			wantMatch:   true,
		},
		"matched, optional minor version": {
			expression1:           "cti.a.p.gr.namespace.v1",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			expr1ParseAsReference: true,
			wantMatch:             true,
		},
		"matched, optional full version": {
			expression1:           "cti.a.p.gr.namespace.v",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			expr1ParseAsReference: true,
			wantMatch:             true,
		},
		"matched, optional minor version in first section": {
			expression1:           "cti.a.p.gr.namespace.v1~a.p.integrations.datacenters.v1.0",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			expr1ParseAsReference: true,
			wantMatch:             true,
		},
		"matched, optional full version in first section": {
			expression1:           "cti.a.p.gr.namespace.v~a.p.integrations.datacenters.v1.0",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			expr1ParseAsReference: true,
			wantMatch:             true,
		},
		"not matched, optional minor version in first section": {
			expression1:           "cti.a.p.gr.namespace.v1~a.p.integrations.datacenters.v1.0",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v2.0",
			expr1ParseAsReference: true,
			wantMatch:             false,
		},
		"not matched, optional full version in first section": {
			expression1:           "cti.a.p.gr.namespace.v~a.p.integrations.datacenters.v1.0",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v2.0",
			expr1ParseAsReference: true,
			wantMatch:             false,
		},
		"not matched, different major version": {
			expression1:           "cti.a.p.gr.namespace.v2.0",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			expr1ParseAsReference: true,
			wantMatch:             false,
		},
		"not matched, different minor version": {
			expression1:           "cti.a.p.gr.namespace.v1.2",
			expression2:           "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
			expr1ParseAsReference: true,
			wantMatch:             false,
		},
		"error, 2nd input contains wildcard in vendor": {
			expression1:     "cti.a.p.gr.namespace.v1.2",
			expression2:     "cti.a.p.gr.namespace.v1.0~*",
			wantMatchErrMsg: "matching against CTI with wildcard is not supported",
		},
		"matched, query, cti": {
			expression1: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0"]`,
			expression2: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0",status="active"]`,
			wantMatch:   true,
		},
		"not matched, query, different attributes": {
			expression1: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0"]`,
			expression2: `cti.a.p.em.event.v1.0[type="cti.a.p.em.topic.v1.0"]`,
			wantMatch:   false,
		},
		"matched, query, raw values": {
			expression1: `cti.a.p.em.event.v1.0[topic="foo",type="bar"]`,
			expression2: `cti.a.p.em.event.v1.0[type="bar",topic="foo"]`,
			wantMatch:   true,
		},
		"not matched, query, different raw values": {
			expression1: `cti.a.p.em.event.v1.0[topic="tenants"]`,
			expression2: `cti.a.p.em.event.v1.0[topic="tenants1"]`,
			wantMatch:   false,
		},
		"matched ignoring query, query, different attributes": {
			expression1: `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0"]`,
			expression2: `cti.a.p.em.event.v1.0[type="cti.a.p.em.topic.v1.0"]`,
			ignoreQuery: true,
			wantMatch:   true,
		},
		"matched ignoring query, query, different raw values": {
			expression1: `cti.a.p.em.event.v1.0[topic="tenants"]`,
			expression2: `cti.a.p.em.event.v1.0[topic="tenants1"]`,
			ignoreQuery: true,
			wantMatch:   true,
		},
		"matched, anonymous expressions": {
			expression1: `cti.a.p.em.event.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05`,
			expression2: `cti.a.p.em.event.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05`,
			wantMatch:   true,
		},
		"not matched, anonymous expressions, different uuid": {
			expression1: `cti.a.p.em.event.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05`,
			expression2: `cti.a.p.em.event.v1.0~3476ff96-3c7a-4984-82d5-22dc5f9955d6`,
			wantMatch:   false,
		},
		"not matched, both expressions are anonymous, different uuid": {
			expression1: `cti.a.p.em.event.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05`,
			expression2: `cti.a.p.em.event.v1.0~3476ff96-3c7a-4984-82d5-22dc5f9955d6`,
			wantMatch:   false,
		},
		"not matched, 1st expression is anonymous, 2nd is not": {
			expression1: `cti.a.p.em.event.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05`,
			expression2: `cti.a.p.em.event.v1.0`,
			wantMatch:   false,
		},
		"matched, inheritance, 2nd expression is anonymous": {
			expression1: `cti.a.p.em.event.v1.0`,
			expression2: `cti.a.p.em.event.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05`,
			wantMatch:   false,
		},
		"not matched, both expressions are anonymous, different inheritance": {
			expression1: `cti.a.p.gr.namespace.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05`,
			expression2: "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0~35a0fdea-077a-4117-ab7a-1eebc309ee05",
			wantMatch:   false,
		},
		"not matched, 1st expression has query, 2nd doesn't have": {
			expression1: "cti.a.p.em.event.v1.0[topic=cti.a.p.em.topic.v1.0~a.p.tasks.v1.0]",
			expression2: "cti.a.p.em.event.v1.0~a.p.task.completed.v1.0",
			ignoreQuery: false,
			wantMatch:   false,
		},
		"matched, 1st expression has query, 2nd doesn't have, but query is ignored": {
			expression1: "cti.a.p.em.event.v1.0[topic=cti.a.p.em.topic.v1.0~a.p.tasks.v1.0]",
			expression2: "cti.a.p.em.event.v1.0~a.p.task.completed.v1.0",
			ignoreQuery: true,
			wantMatch:   true,
		},
	}
	p := NewParser(WithAllowAnonymousEntity(true))
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				exp1     Expression
				parseErr error
			)
			if tt.expr1ParseAsReference {
				exp1, parseErr = p.ParseReference(tt.expression1)
			} else {
				exp1, parseErr = p.Parse(tt.expression1)
			}
			assertNoError(t, parseErr)

			exp2, parseErr := p.Parse(tt.expression2)
			assertNoError(t, parseErr)

			var (
				matchRes bool
				matchErr error
			)

			if tt.ignoreQuery {
				matchRes, matchErr = exp1.MatchIgnoreQuery(exp2)
			} else {
				matchRes, matchErr = exp1.Match(exp2)
			}
			if tt.wantMatchErrMsg != "" {
				assertErrorContains(t, matchErr, tt.wantMatchErrMsg)
				return
			}
			assertNoError(t, matchErr)
			assertEqual(t, tt.wantMatch, matchRes)
		})
	}
}

func Test_Expression_QueryAttributes(t *testing.T) {
	tests := map[string]struct {
		expression            string
		queryAttributeName    AttributeName
		hasQueryAttributes    bool
		isAttributeExist      bool
		isAttributeExpression bool
	}{
		"attribute exist, expression (non cti)": {
			expression:            `cti.a.p.em.event.v1.0[topic="tenant"]`,
			queryAttributeName:    "topic",
			hasQueryAttributes:    true,
			isAttributeExist:      true,
			isAttributeExpression: false,
		},
		"attribute exist, expression": {
			expression:            `cti.a.p.em.event.v1.0[topic="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0"]`,
			queryAttributeName:    "topic",
			hasQueryAttributes:    true,
			isAttributeExist:      true,
			isAttributeExpression: true,
		},
		"attribute not exist": {
			expression:         `cti.a.p.em.event.v1.0[type="cti.a.p.em.topic.v1.0~a.p.tenant.v1.0"]`,
			queryAttributeName: "topic",
			hasQueryAttributes: true,
			isAttributeExist:   false,
		},
		"no query attributes": {
			expression:         `cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0`,
			queryAttributeName: "topic",
			hasQueryAttributes: false,
			isAttributeExist:   false,
		},
	}
	p := &Parser{}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			expr, parseErr := p.Parse(tt.expression)
			assertNoError(t, parseErr)

			assertEqual(t, tt.hasQueryAttributes, expr.HasQueryAttributes())

			attr, exist := expr.GetQueryAttributeValue(tt.queryAttributeName)
			assertEqual(t, tt.isAttributeExist, exist)
			if exist {
				assertEqual(t, tt.isAttributeExpression, attr.IsExpression())
			}
		})
	}
}

func Test_Version(t *testing.T) {
	testCases := map[string]struct {
		version  Version
		expected string
	}{
		"major is wildcard": {
			version:  Version{Major: NullVersion{Value: 1, Valid: true}, Minor: NullVersion{Value: 2, Valid: true}, HasMajorWildcard: true, HasMinorWildcard: true},
			expected: "*",
		},
		"minor is wildcard": {
			version:  Version{Major: NullVersion{Value: 1, Valid: true}, Minor: NullVersion{Value: 2, Valid: true}, HasMajorWildcard: false, HasMinorWildcard: true},
			expected: "1.*",
		},
		"no wildcards": {
			version:  Version{Major: NullVersion{Value: 1, Valid: true}, Minor: NullVersion{Value: 2, Valid: true}, HasMajorWildcard: false, HasMinorWildcard: false},
			expected: "1.2",
		},
		"minor is absent": {
			version:  Version{Major: NullVersion{Value: 1, Valid: true}, Minor: NullVersion{Valid: false}, HasMajorWildcard: false},
			expected: "1",
		},
		"version is absent": {
			version:  Version{},
			expected: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assertEqual(t, tc.expected, tc.version.String())
		})
	}
}
