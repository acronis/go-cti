package filesys

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

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

func OpenZipFile(source string, fpath string) ([]byte, error) {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return nil, fmt.Errorf("open zip file: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != fpath {
			continue
		}

		if file.FileInfo().IsDir() {
			return nil, fmt.Errorf("specified file path %s is a directory", fpath)
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open file in archive: %w", err)
		}
		defer rc.Close()

		return io.ReadAll(rc)
	}

	return nil, fmt.Errorf("failed to find %s in archive", fpath)
}

func UnzipToFS(source string, destination string) ([]string, error) {
	var filenames []string

	reader, err := zip.OpenReader(source)
	if err != nil {
		return filenames, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		fpath := filepath.Join(destination, file.Name)

		// ZipSlip mitigation (https://snyk.io/research/zip-slip-vulnerability)
		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("illegal file path: %s", fpath)
		}
		if strings.Contains(fpath, "..") {
			return filenames, fmt.Errorf("illegal file path: %s", fpath)
		}

		// Prevent symbolic links
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return filenames, fmt.Errorf("symbolic link found: %s", fpath)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return filenames, err
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := file.Open()
		if err != nil {
			return filenames, err
		}

		// Limit the amount of data read to prevent zip bombs
		const maxFileSize = 10 * 1024 * 1024 // 10MB
		_, err = io.CopyN(outFile, rc, maxFileSize+1)
		if err != nil && err != io.EOF {
			return filenames, err
		}
		if err == nil {
			return filenames, fmt.Errorf("file %s exceeds the maximum allowed size", fpath)
		}

		// Close the file without defer before next iteration of loop
		_ = outFile.Close()
		_ = rc.Close()

		if err != io.EOF {
			return filenames, err
		}

		filenames = append(filenames, file.Name)
	}

	return filenames, nil
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
