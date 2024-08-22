package validator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func GetParentCti(cti string) string {
	if pos := strings.LastIndex(cti, "~"); pos != -1 {
		return cti[:pos]
	}
	return cti
}

func GetBaseCti(cti string) string {
	if pos := strings.Index(cti, "~"); pos != -1 {
		return cti[:pos]
	}
	return cti
}

func LoadJsonFile[T interface{}](path string, v *T) error {
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath, _ = filepath.Abs(path)
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}
	return nil
}
