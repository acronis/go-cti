/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package identifier

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const (
	// InheritanceSeparator is a character that separates inheritance parts in CTI expression.
	InheritanceSeparator = '~'

	// Wildcard is a character that represents wildcard in CTI expression.
	Wildcard = '*'
)

// ParseError wraps any parsing error.
type ParseError struct {
	Err           error
	RawExpression string
}

// Error implements "error" interface.
func (e *ParseError) Error() string {
	return e.Err.Error()
}

// Unwrap implements Wrapper interface.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// ErrNotExpression is returned when an input is not a CTI expression.
var ErrNotExpression = errors.New("not CTI expression")

type versionStrategy uint8

const (
	versionStrategyRequireFull versionStrategy = iota
	versionStrategyRequireOnlyMajor
	versionStrategyAllowEmpty
)

type parserParams struct {
	queryDisabled             bool
	attributeSelectorDisabled bool
	versionStrategy           versionStrategy
	wildcardDisabled          bool
}

// Parser is an object for parsing CTI expressions.
type Parser struct {
	allowAnonymousEntity         bool
	allowedDynamicParameterNames []string
}

// ParserOpts represents a parsing options.
type ParserOpts struct {
	AllowedDynamicParameterNames []string
}

// NewParser creates new Parser.
// Available options:
// - WithAllowAnonymousEntity(b bool) - allows parsing anonymous entity UUID in CTI expressions.
// - WithAllowedDynamicParameterNames(names ...string) - allows specifying dynamic parameter names that can be used in CTI expressions.
func NewParser(opts ...ParserOption) *Parser {
	pOpts := makeParserOptions(opts...)
	return &Parser{
		allowAnonymousEntity:         pOpts.allowAnonymousEntity,
		allowedDynamicParameterNames: pOpts.allowedDynamicParameterNames,
	}
}

// Parse parses input string as a CTI expression.
// It accepts all kinds of expressions including identifiers, queries and attribute selectors.
// See ParseQuery, ParseAttributeSelector, ParseIdentifier, ParseReference for more specific parsing.
func Parse(input string, opts ...ParserOption) (Expression, error) {
	return NewParser(opts...).Parse(input)
}

// MustParse parses input string as a CTI expression and panics on error.
// See Parse for more details.
func MustParse(input string, opts ...ParserOption) Expression {
	return NewParser(opts...).MustParse(input)
}

// ParseQuery parses input string as a CTI expression.
// For more details see ParseQuery in Parser.
func ParseQuery(input string, opts ...ParserOption) (Expression, error) {
	return NewParser(opts...).ParseQuery(input)
}

// ParseAttributeSelector parses input string as a CTI expression.
// For more details see ParseAttributeSelector in Parser.
func ParseAttributeSelector(input string, opts ...ParserOption) (Expression, error) {
	return NewParser(opts...).ParseAttributeSelector(input)
}

// ParseIdentifier parses input string as a CTI expression.
// For more details see ParseIdentifier in Parser.
func ParseIdentifier(input string, opts ...ParserOption) (Expression, error) {
	return NewParser(opts...).ParseIdentifier(input)
}

// ParseReference parses input string as a CTI expression.
// For more details see ParseReference in Parser.
func ParseReference(input string, opts ...ParserOption) (Expression, error) {
	return NewParser(opts...).ParseReference(input)
}

// Parse parses input string as a CTI expression.
// It accepts all kinds of expressions including identifiers, references, queries and attribute selectors.
// See ParseQuery, ParseAttributeSelector, ParseIdentifier, ParseReference for more specific parsing.
func (p *Parser) Parse(input string) (Expression, error) {
	return p.parse(input, parserParams{
		queryDisabled:             false,
		attributeSelectorDisabled: false,
		versionStrategy:           versionStrategyRequireFull,
		wildcardDisabled:          false,
	})
}

// ParseQuery parses input string as a CTI expression. It allows only query but not attribute selectors.
// In addition, wildcard is not allowed and optional minor version is allowed.
func (p *Parser) ParseQuery(input string) (Expression, error) {
	return p.parse(input, parserParams{
		queryDisabled:             false,
		attributeSelectorDisabled: true,
		versionStrategy:           versionStrategyRequireOnlyMajor,
		wildcardDisabled:          true,
	})
}

// ParseAttributeSelector parses input string as a CTI expression. It allows only attribute selectors but not queries.
// In addition, wildcard is not allowed and optional minor version is allowed.
// It returns error if attribute selector is absent in input string.
func (p *Parser) ParseAttributeSelector(input string) (Expression, error) {
	expr, err := p.parse(input, parserParams{
		queryDisabled:             true,
		attributeSelectorDisabled: false,
		versionStrategy:           versionStrategyRequireOnlyMajor,
		wildcardDisabled:          true,
	})
	if err != nil {
		return emptyExpression, err
	}
	if expr.AttributeSelector == "" {
		return emptyExpression, fmt.Errorf("attribute selector is absent in input string")
	}
	return expr, nil
}

// ParseIdentifier parses input string as a CTI expression. It allows only identifiers without queries and attribute selectors.
// In addition, wildcard and optional version are not allowed.
func (p *Parser) ParseIdentifier(input string) (Expression, error) {
	return p.parse(input, parserParams{
		queryDisabled:             true,
		attributeSelectorDisabled: true,
		versionStrategy:           versionStrategyRequireFull,
		wildcardDisabled:          true,
	})
}

// ParseReference parses input string as a CTI expression. It allows only identifiers without queries and attribute selectors.
// In addition, wildcards and optional full version are allowed.
func (p *Parser) ParseReference(input string) (Expression, error) {
	return p.parse(input, parserParams{
		queryDisabled:             true,
		attributeSelectorDisabled: true,
		versionStrategy:           versionStrategyAllowEmpty,
		wildcardDisabled:          false,
	})
}

func (p *Parser) parse(input string, params parserParams) (Expression, error) {
	expr, err := p.parseExpression(input, params)
	if err != nil {
		return emptyExpression, &ParseError{Err: err, RawExpression: input}
	}
	return expr, nil
}

// MustParse parses input string as a CTI expression and panics on error.
func (p *Parser) MustParse(input string) Expression {
	expr, err := p.Parse(input)
	if err != nil {
		panic(err)
	}
	return expr
}

func (p *Parser) parseExpression(s string, params parserParams) (Expression, error) {
	if !strings.HasPrefix(s, "cti.") {
		return emptyExpression, ErrNotExpression
	}
	s = s[4:] // cut "cti." prefix

	var err error

	var queryAttributes QueryAttributeSlice
	var attributeSelector AttributeName
	var anonymousEntityUUID uuid.NullUUID

	parseQueryOrSelectorIfPresent := func(s string) (string, error) {
		if !params.queryDisabled {
			if queryAttributes, s, err = p.parseQueryAttributesIfPresent(s); err != nil {
				return s, fmt.Errorf("parse query attributes: %w", err)
			}
		}
		if !params.attributeSelectorDisabled && len(queryAttributes) == 0 {
			if attributeSelector, s, err = p.parseAttributeSelectorIfPresent(s); err != nil {
				return s, fmt.Errorf("parse attribute selector: %w", err)
			}
		}
		return s, nil
	}

	var head *Node
	var tail *Node

	for s != "" {
		if head != nil {
			if tail.HasWildcard() {
				return emptyExpression, fmt.Errorf(`expression may have wildcard "%c" only at the end`, Wildcard)
			}
			if anonymousEntityUUID.Valid {
				return emptyExpression, fmt.Errorf(`expression may have anonymous entity UUID only at the end`)
			}
			if len(queryAttributes) != 0 {
				return emptyExpression, fmt.Errorf(`expression may have query only at the end`)
			}
			if attributeSelector != "" {
				return emptyExpression, fmt.Errorf(`expression may have attribute selector only at the end`)
			}
			if s[0] != InheritanceSeparator {
				return emptyExpression, fmt.Errorf(`expect "%c", got "%c"`, InheritanceSeparator, s[0])
			}
			s = s[1:]
			if s == "" {
				// Dangling separator; e.g. "cti.a.p.gr.namespace.v1.2~"
				return emptyExpression, fmt.Errorf(`unexpected dangling separator "%c"`, InheritanceSeparator)
			}
			if p.allowAnonymousEntity && len(s) >= 36 {
				if anonymousEntityUUID.UUID, err = uuid.Parse(s[:36]); err == nil {
					anonymousEntityUUID.Valid = true
					if s, err = parseQueryOrSelectorIfPresent(s[36:]); err != nil {
						return emptyExpression, err
					}
					continue
				}
			}
		}

		node := &Node{}

		if s[0] == '$' {
			if s, err = p.parseDynamicParameterToNode(s[1:], node); err != nil {
				return emptyExpression, fmt.Errorf("parse dynamic parameter: %w", err)
			}
		} else if s, err = p.parseChunkToNode(s, node, params); err != nil {
			return emptyExpression, err
		}

		if !node.HasWildcard() {
			if s, err = parseQueryOrSelectorIfPresent(s); err != nil {
				return emptyExpression, err
			}
		}

		if head == nil {
			head = node
			tail = node
			continue
		}
		tail.Child = node
		tail = node
	}

	return Expression{
		parser:              p,
		Head:                head,
		QueryAttributes:     queryAttributes,
		AttributeSelector:   attributeSelector,
		AnonymousEntityUUID: anonymousEntityUUID,
	}, nil
}

func (p *Parser) parseDynamicParameterToNode(s string, node *Node) (tail string, err error) {
	if s == "" {
		return s, fmt.Errorf(`expect "{", got end of string`)
	}
	if s[0] != '{' {
		return s, fmt.Errorf(`expect "{", got "%c"`, s[0])
	}
	i := 1
	for i < len(s) && s[i] != '}' {
		i++
	}
	if i == len(s) {
		return s, fmt.Errorf(`expect "}", got end of string`)
	}
	paramName := s[1:i]
	for _, allowedParamName := range p.allowedDynamicParameterNames {
		if paramName == allowedParamName {
			node.DynamicParameterName = paramName
			return s[i+1:], nil
		}
	}
	return s, fmt.Errorf("unknown dynamic parameter %q", paramName)
}

func (p *Parser) parseChunkToNode(s string, node *Node, params parserParams) (tail string, err error) {
	var val string

	// Vendor
	if val, s, err = p.parseVendorOrPackage(s); err != nil {
		return s, fmt.Errorf("parse vendor: %w", err)
	}
	node.Vendor = Vendor(val)
	if node.Vendor.IsWildCard() {
		if params.wildcardDisabled {
			return s, fmt.Errorf("parse vendor: wildcard is disabled")
		}
		return s, nil
	}

	// Package
	if val, s, err = p.parseVendorOrPackage(s); err != nil {
		return s, fmt.Errorf("parse package: %w", err)
	}
	node.Package = Package(val)
	if node.Package.IsWildCard() {
		if params.wildcardDisabled {
			return s, fmt.Errorf("parse package: wildcard is disabled")
		}
		return s, nil
	}

	// EntityName and version
	if node.EntityName, node.Version, s, err = p.parseEntityNameAndVersion(s); err != nil {
		return s, fmt.Errorf("parse entity name and version: %w", err)
	}
	if node.EntityName.EndsWithWildcard() || node.Version.HasWildcard() {
		if params.wildcardDisabled {
			return s, fmt.Errorf("parse entity name and version: wildcard is disabled")
		}
		return s, nil
	}
	if !node.Version.Major.Valid && params.versionStrategy != versionStrategyAllowEmpty {
		return s, fmt.Errorf("parse entity name and version: version is missing")
	}
	if !node.Version.Minor.Valid && params.versionStrategy == versionStrategyRequireFull {
		return s, fmt.Errorf("parse entity name and version: minor part of version is missing")
	}

	return s, nil
}

func (p *Parser) parseVendorOrPackage(s string) (identifier string, tail string, err error) {
	if s == "" {
		return "", "", fmt.Errorf("cannot be empty")
	}

	if s[0] == Wildcard {
		if len(s) > 1 && s[1] != InheritanceSeparator && s[1] != '[' && s[1] != '@' {
			return "", s, fmt.Errorf(`can be "%c" or contain only lower letters, digits, and "_"`, Wildcard)
		}
		return string(Wildcard), s[1:], nil
	}

	i := 0
	for ; i < len(s) && s[i] != '.'; i++ {
		switch {
		case checkByteIsDigit(s[i]) || s[i] == '_':
			if i == 0 {
				return "", s, fmt.Errorf(`can be "%c" or start only with letter`, Wildcard)
			}
		case s[i] >= 'a' && s[i] <= 'z':
		default:
			return "", s, fmt.Errorf(`can be "%c" or contain only lower letters, digits, and "_"`, Wildcard)
		}
	}

	val := s[:i]
	if val == "" {
		return "", s, fmt.Errorf("cannot be empty")
	}

	if i < len(s) && s[i] == '.' {
		i++
	}
	return val, s[i:], nil
}

//nolint:funlen,gocyclo // func implements an alg with well-defined concrete purpose, so high cyclomatic complexity is ok here
func (p *Parser) parseEntityNameAndVersion(s string) (name EntityName, ver Version, tail string, err error) {
	if s == "" {
		return "", Version{}, s, fmt.Errorf(`entity name cannot be empty`)
	}

	majorIdx := -1
	minorIdx := -1
	i := 0
loop:
	for ; i < len(s) && s[i] != InheritanceSeparator && s[i] != '[' && s[i] != '@'; i++ {
		switch {
		case s[i] == '.':
			if i == 0 {
				return "", Version{}, s, fmt.Errorf(`entity name can be "%c" or start only with letter or "_"`, Wildcard)
			}
			if i > 0 && s[i-1] == '.' {
				return "", Version{}, s, fmt.Errorf(`entity name cannot have double dots ("..")`)
			}
			if majorIdx != -1 && minorIdx != -1 {
				majorIdx = -1
				minorIdx = -1
				continue loop
			}
			if majorIdx != -1 && i < len(s)-1 && (s[i+1] == Wildcard || checkByteIsDigit(s[i+1])) {
				minorIdx = i + 1
			}

		case s[i] == '_':
			if i > 0 && s[i-1] == '_' {
				return "", Version{}, s, fmt.Errorf(`entity name cannot have double underscores ("__")`)
			}
			majorIdx = -1
			minorIdx = -1

		case s[i] >= 'a' && s[i] <= 'z':
			if i > 0 && s[i-1] == '.' && s[i] == 'v' && i < len(s)-1 && (s[i+1] == Wildcard || checkByteIsDigit(s[i+1])) {
				majorIdx = i + 1
				minorIdx = -1
				continue loop
			}
			majorIdx = -1
			minorIdx = -1

		case checkByteIsDigit(s[i]):
			if i == 0 {
				return "", Version{}, s, fmt.Errorf(`entity name can be "%c" or start only with letter`, Wildcard)
			}

		case s[i] == Wildcard:
			if i > 0 {
				//nolint:gocritic // if-else if more readable here than nested switch.
				if minorIdx != -1 {
					if minorIdx != i {
						return "", Version{}, s, fmt.Errorf(`minor part of version is invalid`)
					}
				} else if majorIdx != -1 {
					if majorIdx != i {
						return "", Version{}, s, fmt.Errorf(`major part of version is invalid`)
					}
				} else {
					if s[i-1] != '.' {
						return "", Version{}, s, fmt.Errorf(
							`wildcard "%c" in entity name may be only after dot (".")`, Wildcard)
					}
				}
			}
			i++
			break loop

		default:
			return "", Version{}, s, fmt.Errorf(
				`entity name can be "%c" or contain only lower letters, digits, "." and "_"`, Wildcard)
		}
	}

	newS := s[i:]

	if majorIdx == -1 {
		nameStr := s[:i]
		if strings.HasSuffix(nameStr, ".v") {
			nameStr = strings.TrimSuffix(nameStr, ".v")
			if nameStr == "" {
				return "", Version{}, s, fmt.Errorf(`entity name cannot be empty`)
			}
			return EntityName(nameStr), Version{}, newS, nil
		}

		entityName := EntityName(nameStr)
		if !entityName.EndsWithWildcard() {
			return "", Version{}, s, fmt.Errorf("version is missing")
		}
		return entityName, Version{}, newS, nil
	}

	nameStr := s[:majorIdx-2]
	if nameStr == "" {
		return "", Version{}, s, fmt.Errorf(`entity name cannot be empty`)
	}
	entityName := EntityName(nameStr)

	// Parse and validate major version part.
	ver, err = p.parseVersion(s, majorIdx, minorIdx, i)
	if err != nil {
		return "", Version{}, s, err
	}
	return entityName, ver, newS, err
}

func (p *Parser) parseVersion(s string, majorIdx int, minorIdx int, pos int) (Version, error) {
	if s[majorIdx] == Wildcard {
		return Version{HasMajorWildcard: true}, nil
	}

	var minorIsAbsent bool
	if minorIdx == -1 {
		minorIsAbsent = true
		minorIdx = pos + 1
	}

	parseVersionPart := func(s, partName string) (int, error) {
		if s != "" && s[0] == '0' && s != "0" {
			return 0, fmt.Errorf("%s part of version cannot contain leading zero", partName)
		}
		ver, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("parse %s part of version: %w", partName, err)
		}
		if ver < 0 {
			return 0, fmt.Errorf("%s part of version should be >= 0", partName)
		}
		return ver, nil
	}

	majorVerStr := s[majorIdx : minorIdx-1]
	majorVer, err := parseVersionPart(majorVerStr, "major")
	if err != nil {
		return Version{}, err
	}

	if minorIsAbsent {
		return NewPartialVersion(uint(majorVer)), nil
	}

	var minorVer int
	// Parse and validate minor version part
	if s[minorIdx] == Wildcard {
		return Version{Major: NullVersion{uint(majorVer), true}, HasMinorWildcard: true}, nil
	}
	minorVerStr := s[minorIdx:pos]
	minorVer, err = parseVersionPart(minorVerStr, "minor")
	if err != nil {
		return Version{}, err
	}

	if majorVer == 0 && minorVer == 0 {
		return Version{}, fmt.Errorf("version must be higher than 0.0")
	}

	return NewVersion(uint(majorVer), uint(minorVer)), nil
}

func (p *Parser) parseAttributeSelectorIfPresent(s string) (AttributeName, string, error) {
	if s == "" || s[0] != '@' {
		return "", s, nil
	}
	return p.parseAttributeName(s[1:])
}

func (p *Parser) parseQueryAttributesIfPresent(s string) ([]QueryAttribute, string, error) {
	if s == "" || s[0] != '[' {
		return nil, s, nil
	}
	ss := s[1:]

	var res []QueryAttribute

	var queryAttr QueryAttribute
	var err error
	for {
		ss = trimLeftSpaces(ss)
		if ss == "" {
			return nil, s, fmt.Errorf("unexpected end of string")
		}
		if ss[0] == ']' {
			ss = ss[1:]
			break
		}
		if len(res) != 0 {
			if ss[0] != ',' {
				return nil, s, fmt.Errorf(`expect ",", got "%c"`, ss[0])
			}
			ss = trimLeftSpaces(ss[1:])
		}

		queryAttr, ss, err = p.parseQueryAttribute(ss)
		if err != nil {
			return nil, s, err
		}

		for i := range res {
			if res[i].Name == queryAttr.Name {
				return nil, s, fmt.Errorf("non-unique query attribute %q", queryAttr.Name)
			}
		}

		res = append(res, queryAttr)
	}

	if len(res) == 0 {
		return nil, s, fmt.Errorf("query attribute list is empty")
	}

	return res, ss, nil
}

func (p *Parser) parseQueryAttribute(s string) (QueryAttribute, string, error) {
	// Parse attribute name.
	attrName, ss, err := p.parseAttributeName(s)
	if err != nil {
		return QueryAttribute{}, s, err
	}

	// Parse "="
	ss = trimLeftSpaces(ss)
	if ss == "" {
		return QueryAttribute{}, s, fmt.Errorf(`expect "=", got end of string`)
	}
	if ss[0] != '=' {
		return QueryAttribute{}, s, fmt.Errorf(`expect "=", got "%c"`, ss[0])
	}
	ss = trimLeftSpaces(ss[1:])

	// Parse attribute value.
	attrVal, ss, err := p.parseQueryAttributeValue(ss)
	if err != nil {
		return QueryAttribute{}, s, err
	}
	exp, err := p.ParseReference(attrVal)
	if err != nil {
		if errors.Is(err, ErrNotExpression) {
			return QueryAttribute{Name: attrName, Value: QueryAttributeValue{Raw: attrVal}}, ss, nil
		}
		return QueryAttribute{}, s, fmt.Errorf("parse attribute %q as CTI: %w", attrName, err)
	}

	return QueryAttribute{Name: attrName, Value: QueryAttributeValue{Raw: attrVal, Expression: exp}}, ss, nil
}

func (p *Parser) parseAttributeName(s string) (attrName AttributeName, newS string, err error) {
	var i int
loop:
	for ; i < len(s); i++ {
		switch {
		case (s[i] >= '0' && s[i] <= '9') || s[i] == '_':
			if i == 0 || s[i-1] == '.' {
				return "", s, fmt.Errorf("attribute name and its each part should start with letter")
			}
		case s[i] == '.':
			if i == 0 {
				return "", s, fmt.Errorf("attribute name should start with letter")
			}
			if s[i-1] == '.' {
				return "", s, fmt.Errorf(`attribute name cannot have double dots ("..")`)
			}
		case (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z'):
		default:
			break loop
		}
	}
	if i == 0 {
		return "", s, fmt.Errorf(`attribute name cannot be empty and should contain only letters, digits, ".", and "_"`)
	}
	if s[i-1] == '.' {
		return "", s, fmt.Errorf(`attribute name cannot end with dot (".")`)
	}

	return AttributeName(s[:i]), s[i:], nil
}

func (p *Parser) parseQueryAttributeValue(s string) (attrVal string, newS string, err error) {
	if s == "" {
		return "", s, fmt.Errorf(`expect attribute value, got end of string`)
	}
	fn := p.parseQueryAttributeValueNotInQuotes
	if s[0] == '"' || s[0] == '\'' {
		fn = p.parseQueryAttributeValueInQuotes
	}
	return fn(s)
}

func (p *Parser) parseQueryAttributeValueInQuotes(s string) (val string, newS string, err error) {
	quote := s[0]
	escapeCount := 0
	i := 1 // skip quote
	var hasEscapedClosingBracket bool
loop:
	for ; i < len(s); i++ {
		switch s[i] {
		case quote:
			if escapeCount%2 == 1 {
				hasEscapedClosingBracket = true
				escapeCount = 0
				continue loop
			}
			break loop
		case '\\':
			escapeCount++
		default:
			escapeCount = 0
		}
	}
	if i == len(s) {
		return "", s, fmt.Errorf("unexpected end of string while parsing attribute value")
	}
	attrVal := s[1:i]
	if attrVal == "" {
		return "", s, fmt.Errorf("attribute value cannot be empty")
	}
	if hasEscapedClosingBracket {
		attrVal = strings.ReplaceAll(attrVal, string([]byte{'\\', quote}), string([]byte{quote}))
	}
	return attrVal, s[i+1:], nil
}

func (p *Parser) parseQueryAttributeValueNotInQuotes(s string) (val string, newS string, err error) {
	i := 0
	for i < len(s) && s[i] != ',' && s[i] != ']' && s[i] != ' ' {
		i++
	}
	if i == len(s) {
		return "", s, fmt.Errorf("unexpected end of string while parsing attribute value")
	}
	attrVal := s[:i]
	if attrVal == "" {
		return "", s, fmt.Errorf("attribute value cannot be empty")
	}
	return attrVal, s[i:], nil
}

func trimLeftSpaces(s string) string {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	return s[i:]
}

func checkByteIsDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
