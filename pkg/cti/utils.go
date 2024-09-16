package cti

import "strings"

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
