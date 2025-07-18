package validator

import (
	"reflect"
	"testing"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/registry"
)

func TestOnType_Success(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v := New("vendor", "pkg", gr, lr)

	hook := func(v *MetadataValidator, e *metadata.EntityType) error {
		return nil
	}

	err := v.OnType("cti.vendor.pkg.entity_name.v1.0", hook)
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

func TestOnType_ParseError(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v := New("vendor", "pkg", gr, lr)

	// Invalid CTI string should cause parse error
	err := v.OnType("invalid cti", func(v *MetadataValidator, e *metadata.EntityType) error { return nil })
	if err == nil {
		t.Errorf("expected error for invalid CTI, got nil")
	}
}

func TestOnType_TypeHooksCacheInvalidation(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v := New("vendor", "pkg", gr, lr)

	// Simulate initialized typeHooks
	v.typeHooks = map[string][]TypeHook{"dummy": {func(v *MetadataValidator, e *metadata.EntityType) error { return nil }}}

	err := v.OnType("cti.vendor.pkg.entity_name.v1.0", func(v *MetadataValidator, e *metadata.EntityType) error { return nil })
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(v.typeHooks) != 0 {
		t.Errorf("expected typeHooks to be invalidated (empty), got %v", v.typeHooks)
	}
}

func TestOnInstanceOfType_Success(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v := New("vendor", "pkg", gr, lr)

	hook := func(v *MetadataValidator, e *metadata.EntityInstance) error {
		return nil
	}

	err := v.OnInstanceOfType("cti.vendor.pkg.entity_name.v1.0", hook)
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

func TestOnInstanceOfType_ParseError(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v := New("vendor", "pkg", gr, lr)

	err := v.OnInstanceOfType("invalid cti", func(v *MetadataValidator, e *metadata.EntityInstance) error { return nil })
	if err == nil {
		t.Errorf("expected error for invalid CTI, got nil")
	}
}

func TestOnInstanceOfType_InstanceHooksCacheInvalidation(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v := New("vendor", "pkg", gr, lr)

	// Simulate initialized instanceHooks
	v.instanceHooks = map[string][]InstanceHook{"dummy": {func(v *MetadataValidator, e *metadata.EntityInstance) error { return nil }}}

	err := v.OnInstanceOfType("cti.vendor.pkg.entity_name.v1.0", func(v *MetadataValidator, e *metadata.EntityInstance) error { return nil })
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(v.instanceHooks) != 0 {
		t.Errorf("expected instanceHooks to be invalidated (empty), got %v", v.instanceHooks)
	}
}
