package testsupp

import (
	"os"
	"testing"
)

const (
	manualTestInfo = `
	To enable manual tests please define environmental variable GO_MANUAL_TEST. 
	To enable test in VSCode add to settings.json: 
	
		"go.testEnvVars": {
			"GO_MANUAL_TEST": "1"
		},
	`
)

func ManualTest(t *testing.T) {
	t.Helper()
	if os.Getenv("GO_MANUAL_TEST") != "1" {
		t.Skip(manualTestInfo)
	}
}
