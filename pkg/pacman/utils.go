package pacman

import "strings"

func parseGoQuery(goQuery string) (string, string, string) {
	parts := strings.Split(goQuery, " ")
	return parts[0], parts[1], parts[2]
}

func ParseIndexDependency(dep string) (string, string) {
	parts := strings.Split(dep, " ")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
