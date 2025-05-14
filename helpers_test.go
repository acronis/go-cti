/*
Copyright © 2025 Acronis International GmbH.

Released under MIT license.
*/
package cti

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func assertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Values are not equal: expected=%v actual=%v", expected, actual)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func assertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error containing %q, but got nil", substr)
		return
	}
	if !strings.Contains(fmt.Sprint(err), substr) {
		t.Errorf("Expected error containing %q, got: %q", substr, err.Error())
	}
}

func assertEqualError(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error: %q, but got nil", expectedMsg)
		return
	}
	if fmt.Sprint(err) != expectedMsg {
		t.Errorf("Expected error: %q, got: %q", expectedMsg, err.Error())
	}
}

func assertPanicsWithError(t *testing.T, expectedMsg string, f func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Errorf("Expected panic with error: %q, but no panic occurred", expectedMsg)
			return
		}
		if fmt.Sprint(recovered) != expectedMsg {
			t.Errorf("Expected panic with error: %q, got: %v", expectedMsg, recovered)
		}
	}()
	f()
}
