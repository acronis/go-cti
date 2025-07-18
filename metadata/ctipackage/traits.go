package ctipackage

const (
	CTIErrorReportingLegacy = "cti.a.p.dts.func.err_report.v1.0~a.p.legacy.v1.0"
)

type CtiTraitsErrorElem struct {
	// Error corresponds to the JSON schema field "error".
	Error map[string]interface{} `mapstructure:"error"`

	// The error condition. It supports CEL expression and CTI expression. You have
	// access to all the resolvers including $.context, $.params.
	//
	When string `mapstructure:"when"`
}

type CTITrait struct {
	ErrorReporting string               `mapstructure:"error_reporting,omitempty"`
	Error          []CtiTraitsErrorElem `mapstructure:"error,omitempty"`
}
