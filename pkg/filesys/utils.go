package filesys

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/zeebo/xxh3"
)

const (
	AppEnvironVar = "CTIROOT"
	AppUserDir    = ".cti"
)

// GetBaseName Get filename without extension.
func GetBaseName(fileName string) string {
	filename := path.Base(fileName)

	return filename[:len(filename)-len(filepath.Ext(filename))]
}

func GetDirName(filePath string) string {
	return filepath.Base(filepath.Dir(filePath))
}

func ComputeFileChecksum(filePath string) (string, error) {
	slog.Info("Computing checksum", slog.String("path", filePath))
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	hasher := xxh3.Hasher{}
	chunkSize := 1024 * 1024 // 1MB
	buf := make([]byte, 0, chunkSize)
	for {
		nRead, err := f.Read(buf[:chunkSize])
		if nRead == 0 {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read file: %w", err)
		}
		if _, err := hasher.Write(buf[:nRead]); err != nil {
			return "", fmt.Errorf("write hash: %w", err)
		}
	}
	sum128 := hasher.Sum128()
	hexDigest := fmt.Sprintf("%x%x", sum128.Lo, sum128.Hi)
	slog.Info("Checksum created", slog.String("hex", hexDigest))
	return hexDigest, nil
}

func GetRootDir() (string, error) {
	rootDir := os.Getenv(AppEnvironVar)
	if rootDir == "" {
		userDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		rootDir = filepath.Join(userDir, AppUserDir)
	}
	if _, err := os.Stat(rootDir); err != nil {
		err := os.Mkdir(rootDir, 0755)
		if err != nil {
			return "", fmt.Errorf("create root dir: %w", err)
		}
	}
	return rootDir, nil
}

func GetCtiBundlesCacheDir() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", fmt.Errorf("get root dir: %w", err)
	}
	pkgCacheDir := filepath.Join(rootDir, "src")
	if _, err := os.Stat(pkgCacheDir); err != nil {
		if err := os.Mkdir(pkgCacheDir, 0755); err != nil {
			return "", fmt.Errorf("create package cache dir: %w", err)
		}
	}
	return pkgCacheDir, nil
}

func WalkDir(root, ext string) []string {
	var files []string
	filepath.WalkDir(root, func(file string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(dir.Name()) == ext {
			files = append(files, file)
		}
		return nil
	})
	return files
}
