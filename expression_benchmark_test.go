/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package cti

import "testing"

// ---------------------- Benchmarks ----------------------

func Benchmark_Expression_InterpolateDynamicParameterValues(b *testing.B) {
	p := NewParser(WithAllowedDynamicParameterNames("param1", "param2"))
	rawExps := []string{
		"cti.a.p.gr.namespace.v1.0~${param1}~a.p.integrations.cyberdc.v1.1",
		"cti.a.p.gr.namespace.v1.0~${param2}~a.p.integrations.cyberdc.v1.1",
	}
	exps := make([]Expression, len(rawExps))
	var err error
	for i := range rawExps {
		if exps[i], err = p.Parse(rawExps[i]); err != nil {
			b.Fatal(err)
		}
	}

	vals := DynamicParameterValues{
		"param1": "cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.0",
		"param2": "a.p.integrations.datacenters.v1.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err = exps[i%len(exps)].InterpolateDynamicParameterValues(vals); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_Expression_Match(b *testing.B) {
	p := NewParser()
	exp1, err := p.Parse("cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.1~a.p.integrations.cyberdc.v1.*")
	if err != nil {
		b.Fatal(err)
	}
	exp2, err := p.Parse("cti.a.p.gr.namespace.v1.0~a.p.integrations.datacenters.v1.1~a.p.integrations.cyberdc.v1.1")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	var matched bool
	for i := 0; i < b.N; i++ {
		if matched, err = exp1.Match(exp2); err != nil {
			b.Fatal(err)
		}
		if !matched {
			b.Fatal("should be matched")
		}
	}
}
