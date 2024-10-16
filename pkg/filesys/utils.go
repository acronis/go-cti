package filesys

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
)

// GetBaseName Get filename without extension.
func GetBaseName(fileName string) string {
	filename := path.Base(fileName)

	return filename[:len(filename)-len(filepath.Ext(filename))]
}

func GetDirName(filePath string) string {
	return filepath.Base(filepath.Dir(filePath))
}

func CollectFilesWithExt(root, ext string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(root, func(file string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(dir.Name()) == ext {
			files = append(files, file)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}
	return files, nil
}
