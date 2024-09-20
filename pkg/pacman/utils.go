package pacman

import "strings"

func ParseIndexDependency(dep string) (string, string) {
	parts := strings.Split(dep, " ")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
