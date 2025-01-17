/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package cti

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Vendor is a vendor name.
type Vendor string

// IsWildCard returns true if the vendor name is a wildcard.
func (v Vendor) IsWildCard() bool {
	return string(v) == string(Wildcard)
}

// Package is a package name.
type Package string

// IsWildCard returns true if the package name is a wildcard.
func (p Package) IsWildCard() bool {
	return string(p) == string(Wildcard)
}

// EntityName is an entity name.
type EntityName string

// EndsWithWildcard returns true if the entity name contains a wildcard in the end.
func (en EntityName) EndsWithWildcard() bool {
	return en != "" && en[len(en)-1] == Wildcard
}

// NullVersion is a nullable version value.
type NullVersion struct {
	Value uint
	Valid bool // Valid is true if Int32 is not NULL
}

// Version is a version.
type Version struct {
	Major            NullVersion
	HasMajorWildcard bool

	Minor            NullVersion
	HasMinorWildcard bool
}

// NewVersion constructs a new Version.
func NewVersion(major, minor uint) Version {
	return Version{
		Major: NullVersion{major, true},
		Minor: NullVersion{minor, true},
	}
}

// NewPartialVersion constructs a new Version with only major version part.
func NewPartialVersion(major uint) Version {
	return Version{
		Major: NullVersion{major, true},
	}
}

// HasWildcard returns true if either the major or minor part is a wildcard.
func (v Version) HasWildcard() bool {
	return v.HasMajorWildcard || v.HasMinorWildcard
}

// String returns string representation of the Version.
func (v Version) String() string {
	var b strings.Builder
	v.writeToBuilder(&b)
	return b.String()
}

func (v Version) writeToBuilder(b *strings.Builder) {
	// Major
	if v.HasMajorWildcard {
		b.WriteByte(Wildcard)
		return
	}
	if !v.Major.Valid {
		return
	}
	b.WriteString(strconv.FormatUint(uint64(v.Major.Value), 10))
	if !v.Minor.Valid && !v.HasMinorWildcard {
		return
	}

	b.WriteByte('.')

	// Minor
	if v.HasMinorWildcard {
		b.WriteByte(Wildcard)
		return
	}
	b.WriteString(strconv.FormatUint(uint64(v.Minor.Value), 10))
}

// Node represents a parsed complete chunk of CTI expression.
type Node struct {
	Vendor     Vendor
	Package    Package
	EntityName EntityName
	Version    Version

	DynamicParameterName string

	Child *Node
}

// AttributeName is a name of the attribute that may be used in CTI query and attribute selector.
type AttributeName string

// QueryAttributeSlice is a slice of QueryAttribute.
type QueryAttributeSlice []QueryAttribute

// Match reports true if QueryAttributeSlice matches with the second QueryAttributeSlice.
func (as QueryAttributeSlice) Match(attrSlice2 QueryAttributeSlice) (bool, error) {
	for i := range as {
		queryAttr1 := &as[i]

		var queryAttr2 *QueryAttribute
		for j := range attrSlice2 {
			if attrSlice2[j].Name == queryAttr1.Name {
				queryAttr2 = &attrSlice2[j]
				break
			}
		}
		if queryAttr2 == nil {
			return false, nil
		}

		if !queryAttr1.Value.IsExpression() && !queryAttr2.Value.IsExpression() {
			if queryAttr1.Value.Raw != queryAttr2.Value.Raw {
				return false, nil
			}
			continue
		}
		if !queryAttr1.Value.IsExpression() || !queryAttr2.Value.IsExpression() {
			return false, nil
		}
		queryAttrMatched, queryMatchErr := queryAttr1.Value.Expression.Match(queryAttr2.Value.Expression)
		if queryMatchErr != nil {
			return false, fmt.Errorf("match query attribute %q: %w", queryAttr1.Name, queryMatchErr)
		}
		if !queryAttrMatched {
			return false, nil
		}
	}
	return true, nil
}

// QueryAttribute is an attribute that is used in CTI query.
type QueryAttribute struct {
	Name  AttributeName
	Value QueryAttributeValue
}

// QueryAttributeValue is value of the attribute that is used in CTI query.
type QueryAttributeValue struct {
	Raw        string
	Expression Expression
}

// IsExpression return true if QueryAttributeValue is expression
func (v QueryAttributeValue) IsExpression() bool {
	return v.Expression.Head != nil || len(v.Expression.QueryAttributes) != 0
}

// HasWildcard returns true if Node contains wildcard in any section.
func (n *Node) HasWildcard() bool {
	return n.Vendor.IsWildCard() || n.Package.IsWildCard() || n.EntityName.EndsWithWildcard() ||
		n.Version.HasMajorWildcard || n.Version.HasMinorWildcard
}

// HasDynamicParameters returns true if the Node contains a dynamic parameter.
func (n *Node) HasDynamicParameters() bool {
	return n.DynamicParameterName != ""
}

// String returns string representation of the Node.
func (n *Node) String() string {
	b := strings.Builder{}
	n.writeToBuilder(&b)
	return b.String()
}

func (n *Node) writeToBuilder(b *strings.Builder) {
	if n.DynamicParameterName != "" {
		b.WriteByte('$')
		b.WriteString(n.DynamicParameterName)
		return
	}

	b.WriteString(string(n.Vendor))
	if n.Vendor.IsWildCard() {
		return
	}

	b.WriteByte('.')
	b.WriteString(string(n.Package))
	if n.Package.IsWildCard() {
		return
	}

	b.WriteByte('.')
	b.WriteString(string(n.EntityName))
	if n.EntityName.EndsWithWildcard() {
		return
	}

	b.WriteByte('.')
	b.WriteByte('v')
	n.Version.writeToBuilder(b)
}

// Expression represents a parsed CTI expression.
type Expression struct {
	parser *Parser

	// Head is a head node of the expression.
	// Each node contains a complete chunk of the expression (cti.<vendor>.<package>.<entity>.v<major>.<minor>).
	Head *Node

	// QueryAttributes is a slice of query attributes.
	// For example, for the following CTI expression:
	// cti.a.p.am.alert.v1.0~a.p.activity.canceled.v1.0[category="cti.a.p.am.category.v1.0~a.p.backup.v1.0",severity="critical"]
	// QueryAttributes will contain two attributes ("category" and "severity")
	// with values ("cti.a.p.am.category.v1.0~a.p.backup.v1.0" and "critical").
	QueryAttributes QueryAttributeSlice

	// AttributeSelector is a selector of the attribute.
	// For example, for the following CTI expression:
	// cti.a.p.am.alert.v1.0~a.p.activity.canceled.v1.0@category
	// AttributeSelector will contain "category".
	AttributeSelector AttributeName

	// AnonymousEntityUUID is an anonymous entity UUID.
	// For example, for the following CTI expression:
	// cti.a.p.am.alert.v1.0~ba3c448e-55e3-4f7f-ae54-4e87eb8635f6
	// AnonymousEntityUUID will contain "ba3c448e-55e3-4f7f-ae54-4e87eb8635f6",
	// and will be marked as valid (AnonymousEntityUUID.Valid == true).
	AnonymousEntityUUID uuid.NullUUID
}

var emptyExpression = Expression{}

// Tail returns the last node.
func (e *Expression) Tail() *Node {
	n := e.Head
	for n != nil && n.Child != nil {
		n = n.Child
	}
	return n
}

// HasWildcard returns true if the Expression contains wildcard.
func (e *Expression) HasWildcard() bool {
	for n := e.Head; n != nil; n = n.Child {
		if n.HasWildcard() {
			return true
		}
	}
	return false
}

// HasAnonymousEntity returns true if the Expression contains an anonymous entity.
func (e *Expression) HasAnonymousEntity() bool {
	return e.AnonymousEntityUUID.Valid
}

// HasQueryAttributes returns true if the Expression contains any query attributes.
func (e *Expression) HasQueryAttributes() bool {
	return len(e.QueryAttributes) != 0
}

// HasDynamicParameters returns true if the Expression contains a dynamic parameter in any node or query attribute.
func (e *Expression) HasDynamicParameters() bool {
	for n := e.Head; n != nil; n = n.Child {
		if n.HasDynamicParameters() {
			return true
		}
	}
	for i := range e.QueryAttributes {
		queryAttr := &e.QueryAttributes[i]
		if queryAttr.Value.IsExpression() && queryAttr.Value.Expression.HasDynamicParameters() {
			return true
		}
	}
	return false
}

// GetQueryAttributeValue returns QueryAttributeValue of the Expression by the attribute name.
func (e *Expression) GetQueryAttributeValue(name AttributeName) (QueryAttributeValue, bool) {
	for i := range e.QueryAttributes {
		if e.QueryAttributes[i].Name == name {
			return e.QueryAttributes[i].Value, true
		}
	}
	return QueryAttributeValue{}, false
}

// String returns the string representation of the whole CTI expression.
func (e *Expression) String() string {
	res := strings.Builder{}
	for node := e.Head; node != nil; node = node.Child {
		if res.Len() == 0 {
			res.WriteString("cti.")
		} else {
			res.WriteByte(InheritanceSeparator)
		}
		node.writeToBuilder(&res)
	}

	if e.AnonymousEntityUUID.Valid {
		res.WriteByte('~')
		res.WriteString(e.AnonymousEntityUUID.UUID.String())
	}

	if len(e.QueryAttributes) != 0 {
		res.WriteByte('[')
		for i := range e.QueryAttributes {
			if i > 0 {
				res.WriteByte(',')
			}
			res.WriteString(string(e.QueryAttributes[i].Name))
			res.WriteByte('=')
			res.WriteByte('"')
			attrVal := e.QueryAttributes[i].Value.Raw
			if e.QueryAttributes[i].Value.IsExpression() {
				attrVal = e.QueryAttributes[i].Value.Expression.String()
			}
			res.WriteString(strings.ReplaceAll(attrVal, "\"", "\\\""))
			res.WriteByte('"')
		}
		res.WriteByte(']')
	}

	if e.AttributeSelector != "" {
		res.WriteByte('@')
		res.WriteString(string(e.AttributeSelector))
	}

	return res.String()
}

// Match reports whether the Expression contains any match of the second expression.
func (e *Expression) Match(secondExpression Expression) (bool, error) {
	return e.match(secondExpression, false)
}

// MatchIgnoreQuery reports whether the Expression contains any match of the second expression (ignoring expression query).
func (e *Expression) MatchIgnoreQuery(secondExpression Expression) (bool, error) {
	return e.match(secondExpression, true)
}

//nolint:gocyclo // func implements an alg with well-defined concrete purpose, so high cyclomatic complexity is ok here
func (e *Expression) match(secondExpression Expression, ignoreQuery bool) (bool, error) {
	if e.AttributeSelector != "" {
		return false, fmt.Errorf("matching of CTI with attribute selector is not supported")
	}
	if secondExpression.AttributeSelector != "" {
		return false, fmt.Errorf("matching against CTI with attribute selector is not supported")
	}
	if secondExpression.HasWildcard() {
		return false, fmt.Errorf("matching against CTI with wildcard is not supported")
	}

	curNode1 := e.Head
	curNode2 := secondExpression.Head
	for ; curNode1 != nil && curNode2 != nil; curNode1, curNode2 = curNode1.Child, curNode2.Child {
		// Vendor matching.
		if curNode1.Vendor.IsWildCard() {
			return true, nil
		}
		if curNode1.Vendor != curNode2.Vendor {
			return false, nil
		}

		// Package matching.
		if curNode1.Package.IsWildCard() {
			return true, nil
		}
		if curNode1.Package != curNode2.Package {
			return false, nil
		}

		// Entity type matching.
		if curNode1.EntityName.EndsWithWildcard() {
			entityName1Prefix := string(curNode1.EntityName)
			entityName1Prefix = entityName1Prefix[:len(entityName1Prefix)-1] // Remove wildcard from the end.
			// Prefix contains dot in the end after wildcard removal,
			// so we need to add it to the second entity name as well for matching,
			// since the version goes right after entity name.
			if !strings.HasPrefix(string(curNode2.EntityName)+".", entityName1Prefix) {
				return false, nil
			}
			return true, nil
		}
		if curNode1.EntityName != curNode2.EntityName {
			return false, nil
		}

		// Entity version matching.
		if curNode1.Version.HasMajorWildcard {
			return true, nil
		}
		if !curNode1.Version.Major.Valid {
			continue
		}
		if curNode1.Version.Major != curNode2.Version.Major {
			return false, nil
		}
		if curNode1.Version.HasMinorWildcard {
			return true, nil
		}
		if !curNode1.Version.Minor.Valid {
			continue
		}
		if curNode1.Version.Minor != curNode2.Version.Minor {
			return false, nil
		}
	}

	switch {
	case curNode1 == nil && curNode2 == nil:
		if e.AnonymousEntityUUID != secondExpression.AnonymousEntityUUID {
			return false, nil
		}
		if !ignoreQuery {
			if qaMatched, err := e.QueryAttributes.Match(secondExpression.QueryAttributes); err != nil || !qaMatched {
				return false, err
			}
		}

	case curNode1 != nil: // curNode2 == nil
		return false, nil

	default: // curNode2 != nil && curNode1  == nil
		if e.AnonymousEntityUUID.Valid || (!ignoreQuery && e.HasQueryAttributes()) {
			return false, nil
		}
	}

	return true, nil
}

// DynamicParameterValues is a container (map) of dynamic parameter values that can be interpolated into the Expression.
type DynamicParameterValues map[string]string

// InterpolateDynamicParameterValues interpolates dynamic parameter values into the Expression.
//
//nolint:funlen // func implements an alg with well-defined concrete purpose, so high cyclomatic complexity is ok here
func (e *Expression) InterpolateDynamicParameterValues(values DynamicParameterValues) (Expression, error) {
	var cpHead *Node
	var cpPrevNode *Node

	initHeadAndPrevNodes := func(n *Node) {
		if cpHead == nil {
			cpHead = n
			cpPrevNode = n
			return
		}
		cpPrevNode.Child = n
		cpPrevNode = n
	}

	for curNode := e.Head; curNode != nil; curNode = curNode.Child {
		if curNode.DynamicParameterName == "" {
			initHeadAndPrevNodes(&Node{
				Vendor:     curNode.Vendor,
				Package:    curNode.Package,
				EntityName: curNode.EntityName,
				Version:    curNode.Version,
			})
			continue
		}

		val, ok := values[curNode.DynamicParameterName]
		if !ok {
			return emptyExpression, fmt.Errorf("dynamic parameter values do not have %q key", curNode.DynamicParameterName)
		}

		valToParse := val
		isCompleteCTI := true
		if !strings.HasPrefix(val, "cti.") {
			valToParse = "cti." + val
			isCompleteCTI = false
		}

		// Parse value as CTI expression.
		p := *e.parser
		p.allowedDynamicParameterNames = nil // avoid recursion
		parsedExp, parseErr := p.Parse(valToParse)
		if parseErr != nil {
			return emptyExpression, fmt.Errorf("parse value %q of dynamic parameter %q as CTI: %w",
				val, curNode.DynamicParameterName, parseErr)
		}

		if !isCompleteCTI {
			initHeadAndPrevNodes(parsedExp.Head)
			continue
		}

		if cpHead != nil {
			prefixExp := Expression{Head: cpHead}
			match, matchErr := prefixExp.Match(parsedExp)
			if matchErr != nil {
				return emptyExpression, fmt.Errorf("match %q and value %q of dynamic parameter %q: %w",
					prefixExp.String(), val, curNode.DynamicParameterName, matchErr)
			}
			if !match {
				return emptyExpression, fmt.Errorf("%q and value %q of dynamic parameter %q are not matched",
					prefixExp.String(), val, curNode.DynamicParameterName)
			}
		}
		cpHead = parsedExp.Head
		cpPrevNode = parsedExp.Tail()
	}

	cpQueryAttributes := make([]QueryAttribute, len(e.QueryAttributes))
	for i := range e.QueryAttributes {
		queryAttr := &e.QueryAttributes[i]
		var cpExp Expression
		if queryAttr.Value.IsExpression() {
			var err error
			if cpExp, err = queryAttr.Value.Expression.InterpolateDynamicParameterValues(values); err != nil {
				return emptyExpression, fmt.Errorf("interpolate dynamic parameters for attribute %q: %w",
					queryAttr.Name, err)
			}
		}
		cpQueryAttributes[i] = QueryAttribute{
			Name: queryAttr.Name,
			Value: QueryAttributeValue{
				Raw:        queryAttr.Value.Raw,
				Expression: cpExp,
			},
		}
	}

	return Expression{Head: cpHead, QueryAttributes: cpQueryAttributes}, nil
}
