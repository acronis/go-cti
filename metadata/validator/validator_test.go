package validator

import (
	"reflect"
	"testing"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/registry"
	"github.com/stretchr/testify/require"
)

func Test_onType_Success(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v, err := New("vendor", "pkg", gr, lr)
	require.NoError(t, err)

	hook := func(v *MetadataValidator, e *metadata.EntityType) error {
		return nil
	}

	err = v.onType("cti.vendor.pkg.entity_name.v1.0", hook)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be registered in aggregateTypeHooks
	found := false
	for _, hooks := range v.aggregateTypeHooks {
		for _, h := range hooks {
			// Compare function pointers using reflect
			if reflect.ValueOf(h).Pointer() == reflect.ValueOf(hook).Pointer() {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("hook not registered in aggregateTypeHooks")
	}
}

func Test_onType_ParseError(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v, err := New("vendor", "pkg", gr, lr)
	require.NoError(t, err)

	// Invalid CTI string should cause parse error
	err = v.onType("invalid cti", func(v *MetadataValidator, e *metadata.EntityType) error { return nil })
	if err == nil {
		t.Errorf("expected error for invalid CTI, got nil")
	}
}

func Test_onInstanceOfType_Success(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v, err := New("vendor", "pkg", gr, lr)
	require.NoError(t, err)

	hook := func(v *MetadataValidator, e *metadata.EntityInstance) error {
		return nil
	}

	err = v.onInstanceOfType("cti.vendor.pkg.entity_name.v1.0", hook)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be registered in aggregateInstanceHooks
	found := false
	for _, hooks := range v.aggregateInstanceHooks {
		for _, h := range hooks {
			if reflect.ValueOf(h).Pointer() == reflect.ValueOf(hook).Pointer() {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("hook not registered in aggregateInstanceHooks")
	}
}

func Test_onInstanceOfType_ParseError(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v, err := New("vendor", "pkg", gr, lr)
	require.NoError(t, err)

	err = v.onInstanceOfType("invalid cti", func(v *MetadataValidator, e *metadata.EntityInstance) error { return nil })
	if err == nil {
		t.Errorf("expected error for invalid CTI, got nil")
	}
}
