package pacman

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/acronis/go-cti/metadata/ctipackage"
)

const (
	RAMLExt = ".raml"
)

var patchDepsRe = regexp.MustCompile(fmt.Sprintf(`(?:..\/)*(%s)\/`, strings.ReplaceAll(ctipackage.DependencyDirName, ".", `\.`)))
var patchRamlxRe = regexp.MustCompile(fmt.Sprintf(`(?:..\/)*(%s)\/`, strings.ReplaceAll(ctipackage.RamlxDirName, ".", `\.`)))

func replaceCaptureGroup(reg *regexp.Regexp, input, replacement string, groupIndex int) string {
	matches := reg.FindAllStringSubmatchIndex(input, -1)
	if matches == nil {
		return input
	}

	sb := strings.Builder{}
	prevEnd := 0
	for _, match := range matches {
		sb.WriteString(input[prevEnd:match[2*groupIndex]])
		sb.WriteString(replacement)
		prevEnd = match[2*groupIndex+1]
	}
	sb.WriteString(input[prevEnd:])

	return sb.String()
}

func patchRelativeLinks(dir string) error {
	if err := filepath.WalkDir(dir, func(file string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(dir.Name()) != RAMLExt {
			return nil
		}

		raw, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		// implies that dependencies are correct in the first place
		content := replaceCaptureGroup(patchDepsRe, string(raw), "../../"+ctipackage.DependencyDirName, 1)
		content = replaceCaptureGroup(patchRamlxRe, content, "../../"+ctipackage.RamlxDirName, 1)

		if err = os.WriteFile(file, []byte(content), 0600); err != nil {
			return fmt.Errorf("patch .raml file: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}
	return nil
}
