package filesys

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/rogpeppe/go-internal/dirhash"
	"github.com/zeebo/xxh3"
)

func hashXXH3(files []string, open func(string) (io.ReadCloser, error)) (string, error) {
	h := xxh3.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return "", errors.New("dirhash: filenames with newlines are not supported")
		}
		r, err := open(file)
		if err != nil {
			return "", err
		}
		hf := xxh3.New()
		_, err = io.Copy(hf, r)
		r.Close()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), file)
	}
	return "xxh3:" + base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func ComputeFileChecksum(filePath string) (string, error) {
	return hashXXH3([]string{filePath}, func(name string) (io.ReadCloser, error) {
		return os.Open(name)
	})
}

func ComputeDirectoryHash(dir string) (string, error) {
	return dirhash.HashDir(dir, "", hashXXH3)
}
