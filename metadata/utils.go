package metadata

import "strings"

// GetParentCTI returns the parent CTI of the given CTI string.
// The parent CTI is the part before the last "~" character.
// If there is no "~", it returns an empty string, indicating no direct parent.
func GetParentCTI(cti string) string {
	if pos := strings.LastIndex(cti, "~"); pos != -1 {
		return cti[:pos]
	}
	return "" // Empty string means no direct parent
}

// GetBaseCTI returns the base CTI of the given CTI string.
// The base CTI is the part before the first "~" character.
// If there is no "~", it returns the original CTI.
func GetBaseCTI(cti string) string {
	if pos := strings.Index(cti, "~"); pos != -1 {
		return cti[:pos]
	}
	return cti
}
