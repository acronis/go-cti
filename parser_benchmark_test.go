/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package cti

import (
	"crypto/md5"
	"regexp"
	"testing"
)

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

func Benchmark_Parse_Identifier(b *testing.B) {
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(benchParseExprIdentifiers[i%len(benchParseExprIdentifiers)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_Parse_Wildcard(b *testing.B) {
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(benchParseExprWildcards[i%len(benchParseExprWildcards)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_Parse_Query(b *testing.B) {
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

func Benchmark_Parse_IdentifierRegExp(b *testing.B) {
	// RegExp source: https://github.com/acronis/go-cti/blob/main/metadata/ramlx/spec_v1/cti.raml#L116
	regExp := regexp.MustCompile(`^cti\.([a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z_][a-z0-9_.]*\.v[\d]+\.[\d]+)(~([a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z_][a-z0-9_.]*\.v[\d]+\.[\d]+))*(~[0-9a-f]{8}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{12})?$`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := benchParseExprIdentifiers[i%len(benchParseExprIdentifiers)]
		if !regExp.MatchString(s) {
			b.Fatalf("%q is not matched", s)
		}
	}
}

func Benchmark_Parse_WildcardRegExp(b *testing.B) {
	// RegExp source: https://github.com/acronis/go-cti/blob/main/metadata/ramlx/spec_v1/cti.raml#L141
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
func Benchmark_MD5(b *testing.B) {
	expBytes := make([][]byte, len(benchParseExprIdentifiers))
	for i := range expBytes {
		expBytes[i] = []byte(benchParseExprIdentifiers[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = md5.Sum(expBytes[i%len(expBytes)])
	}
}
