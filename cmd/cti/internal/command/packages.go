package command

import (
	"fmt"
	"strings"
)

func ParsePackages(args []string) (map[string]string, error) {
	pkgs := map[string]string{}
	for _, pkg := range args {
		chunks := strings.Split(pkg, "@")
		if len(chunks) != 2 {
			return nil, fmt.Errorf("invalid package format: %s, should be `<source>@<version>`", pkg)
		}
		if _, ok := pkgs[chunks[0]]; ok {
			return nil, fmt.Errorf("duplicate package: %s", chunks[0])
		}
		pkgs[chunks[0]] = chunks[1]
	}

	return pkgs, nil
}
