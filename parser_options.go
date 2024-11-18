/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package cti

// ParserOption is an interface for functional options that can be passed to the NewParser constructor.
type ParserOption interface {
	apply(*parserOptions)
}

type parserOptions struct {
	allowAnonymousEntity         bool
	allowedDynamicParameterNames []string
}

type allowAnonymousEntityParserOption bool

func (o allowAnonymousEntityParserOption) apply(opts *parserOptions) {
	opts.allowAnonymousEntity = bool(o)
}

// WithAllowAnonymousEntity allows specifying whether the anonymous entity is allowed to be used in the CTI.
func WithAllowAnonymousEntity(b bool) ParserOption {
	return allowAnonymousEntityParserOption(b)
}

type allowedDynamicParameterNamesParserOption []string

func (o allowedDynamicParameterNamesParserOption) apply(opts *parserOptions) {
	opts.allowedDynamicParameterNames = o
}

// WithAllowedDynamicParameterNames allows specifying dynamic parameter names that are allowed to be used in the CTI.
func WithAllowedDynamicParameterNames(names ...string) ParserOption {
	return allowedDynamicParameterNamesParserOption(names)
}

func makeParserOptions(opts ...ParserOption) parserOptions {
	var options parserOptions
	for _, opt := range opts {
		opt.apply(&options)
	}
	return options
}
