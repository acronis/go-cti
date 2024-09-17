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

// GetBaseName Get filename without extension.
func GetBaseName(fileName string) string {
	filename := path.Base(fileName)
	name := filename[:len(filename)-len(filepath.Ext(filename))]

	return name
}

func GetDirName(filePath string) string {
	return filepath.Base(filepath.Dir(filePath))
}

func ComputeFileHexdigest(filePath string) (string, error) {
	slog.Info(fmt.Sprintf("Computing hash of %s", filePath))
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
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
			return "", err
		}
		hasher.Write(buf[:nRead])
	}
	sum128 := hasher.Sum128()
	hexdigest := fmt.Sprintf("%x%x", sum128.Lo, sum128.Hi)
	slog.Info(fmt.Sprintf("Computed file hexdigest: %s", hexdigest))
	return hexdigest, nil
}

func GetRootDir() (string, error) {
	rootDir := os.Getenv("CTIROOT")
	if rootDir == "" {
		userDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		rootDir = filepath.Join(userDir, ".cti")
	}
	if _, err := os.Stat(rootDir); err != nil {
		os.Mkdir(rootDir, 0755)
	}
	return rootDir, nil
}

func GetPkgCacheDir() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", err
	}
	pkgCacheDir := filepath.Join(rootDir, "src")
	if _, err := os.Stat(pkgCacheDir); err != nil {
		os.Mkdir(pkgCacheDir, 0755)
	}
	return pkgCacheDir, nil
}

func OpenZipFile(source string, fpath string) ([]byte, error) {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return nil, err
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
			return nil, err
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
