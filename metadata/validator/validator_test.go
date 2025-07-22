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

	hook := func(v *MetadataValidator, e *metadata.EntityType, _ any) error {
		return nil
	}

	err = v.onType(TypeRule{
		ID:             "test_rule",
		ValidationHook: hook,
		Expression:     "cti.vendor.pkg.entity_name.v1.0",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be registered in aggregateTypeRules
	found := false
	for _, hooks := range v.aggregateTypeRules {
		for _, h := range hooks {
			// Compare function pointers using reflect
			if reflect.ValueOf(h.ValidationHook).Pointer() == reflect.ValueOf(hook).Pointer() {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("hook not registered in aggregateTypeRules")
	}
}

func Test_onType_ParseError(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v, err := New("vendor", "pkg", gr, lr)
	require.NoError(t, err)

	// Invalid CTI string should cause parse error
	err = v.onType(TypeRule{
		ID:             "test_rule",
		ValidationHook: func(v *MetadataValidator, e *metadata.EntityType, _ any) error { return nil },
		Expression:     "invalid cti",
	})
	if err == nil {
		t.Errorf("expected error for invalid CTI, got nil")
	}
}

func Test_onInstanceOfType_Success(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v, err := New("vendor", "pkg", gr, lr)
	require.NoError(t, err)

	hook := func(v *MetadataValidator, e *metadata.EntityInstance, _ any) error {
		return nil
	}

	err = v.onInstanceOfType(InstanceRule{
		ID:             "test_rule",
		ValidationHook: hook,
		Expression:     "cti.vendor.pkg.entity_name.v1.0",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be registered in aggregateInstanceRules
	found := false
	for _, hooks := range v.aggregateInstanceRules {
		for _, h := range hooks {
			if reflect.ValueOf(h.ValidationHook).Pointer() == reflect.ValueOf(hook).Pointer() {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("hook not registered in aggregateInstanceRules")
	}
}

func Test_onInstanceOfType_ParseError(t *testing.T) {
	gr := registry.New()
	lr := registry.New()
	v, err := New("vendor", "pkg", gr, lr)
	require.NoError(t, err)

	err = v.onInstanceOfType(InstanceRule{
		ID:             "test_rule",
		ValidationHook: func(v *MetadataValidator, e *metadata.EntityInstance, _ any) error { return nil },
		Expression:     "invalid cti",
	})
	if err == nil {
		t.Errorf("expected error for invalid CTI, got nil")
	}
}
