package downloader

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/filesys"
)

var (
	goImportRe = regexp.MustCompile("<meta name=\"go-import\" content=\"([^\"]+)")
	wsRe       = regexp.MustCompile(`\s+`)
)

type Downloader interface {
	// Download downloads dependencies and returns a list of installed dependencies
	// and a list of replaced dependencies.
	Download(depends []string, replace bool) ([]string, map[string]struct{}, error)
}

type goLikeDownloader struct {
	indexLock       *bundle.IndexLock
	cacheDir        string
	dependenciesDir string
}

func New(indexLock *bundle.IndexLock, cacheDir string, dependsDir string) Downloader {
	return &goLikeDownloader{
		indexLock:       indexLock,
		cacheDir:        cacheDir,
		dependenciesDir: dependsDir,
	}
}

func (dl *goLikeDownloader) Download(depends []string, replace bool) ([]string, map[string]struct{}, error) {
	var replaced = make(map[string]struct{})
	var installed []string
	for _, dep := range depends {
		// TODO: Dependency consists of two space-delimited parts:
		// 1. Dependency name
		// 2. Dependency version
		sourceName, depVersion := bundle.ParseIndexDependency(dep)
		if depVersion == "" {
			depVersion = "main"
		}

		source := fmt.Sprintf("https://%s", sourceName)
		body, err := dl.discoverSource(source)
		if err != nil {
			return nil, nil, fmt.Errorf("discover source at %s: %w", source, err)
		}

		m := goImportRe.FindStringSubmatch(string(body))
		if len(m) == 0 {
			return nil, nil, fmt.Errorf("find go-import at %s", source)
		}
		slog.Info("Discovered dependency", slog.String("bundle", sourceName))
		_, _, sourceLocation := dl.parseGoQuery(m[len(m)-1])

		// FIXME: This will only work with git source!
		commitHash, err := gitLsRemote(sourceLocation, depVersion)
		if err != nil {
			return nil, nil, err
		}
		if commitHash == "" {
			return nil, nil, fmt.Errorf("failed to find %s %s", sourceName, depVersion)
		}

		if b, ok := dl.indexLock.Bundles[sourceName]; ok {
			// TODO: Bundle version comparison using semver?
			_, err := os.Stat(filepath.Join(dl.dependenciesDir, b.AppCode))
			if b.Integrity == commitHash && err == nil {
				slog.Info("Bundle was not changed. Skipping.")
				continue
			}
		}

		// go-import consists of space-delimited data with:
		// 1. Dependency name
		// 2. Source type (mod, vcs, git)
		// 3. Source location
		// TODO: Support other source types?
		cacheZip, err := dl.loadGitDependency(sourceName, sourceLocation, depVersion, commitHash)
		if err != nil {
			return nil, nil, err
		}

		rc, err := filesys.OpenZipFile(cacheZip, bundle.IndexFileName)
		if err != nil {
			return nil, nil, fmt.Errorf("open index.json in %s: %w", cacheZip, err)
		}
		depIdx, err := bundle.UnmarshalIndexFile(rc)
		if err != nil {
			return nil, nil, fmt.Errorf("unmarshal index.json in %s: %w", cacheZip, err)
		}

		if depIdx.AppCode == "" {
			return nil, nil, fmt.Errorf("empty application code in %s", sourceName)
		}

		// TODO: Comparing against the commit hash instead? This is dependent on the source type...
		// hexdigest, err := utils.ComputeFileHexdigest(cacheZip)
		// if err != nil {
		// 	return err
		// }

		// TODO: This probably should not be allowed for indirect dependencies as it would switch dependency back and forth
		if s, ok := dl.indexLock.Sources[depIdx.AppCode]; ok && s.Source != sourceName {
			slog.Warn(fmt.Sprintf("%s was already installed from %s.", depIdx.AppCode, s.Source))
			if !replace {
				continue
			}
			slog.Warn(fmt.Sprintf("Replacing %s with %s.", s.Source, sourceName))
			delete(dl.indexLock.Bundles, s.Source)
			replaced[s.Source] = struct{}{}
		}

		dest := filepath.Join(dl.dependenciesDir, depIdx.AppCode)
		if _, err := os.Stat(dest); err == nil {
			if err = os.RemoveAll(dest); err != nil {
				return nil, nil, err
			}
		}

		if err := filesys.SecureUnzip(cacheZip, dest); err != nil {
			return nil, nil, err
		}

		dl.indexLock.Sources[depIdx.AppCode] = bundle.SourceInfo{
			Source: sourceName,
		}

		dl.indexLock.Bundles[sourceName] = bundle.Info{
			Name:      "",
			AppCode:   depIdx.AppCode,
			Integrity: commitHash,
			Version:   depVersion, // TODO: Use golang pseudo-version format
			Source:    source,
			Depends:   depIdx.Depends,
		}

		if depIdx.Depends != nil {
			depInstalled, depReplaced, err := dl.Download(depIdx.Depends, replace)
			if err != nil {
				return nil, nil, err
			}
			installed = append(installed, depInstalled...)
			for k := range depReplaced {
				replaced[k] = struct{}{}
			}
		}

		installed = append(installed, sourceName)
	}
	return installed, replaced, nil
}

func (dl *goLikeDownloader) loadGitDependency(sourceName string, source string, ref string, hash string) (string, error) {
	filename := fmt.Sprintf("%s-%s-%s.zip", filepath.Base(sourceName), ref, hash[:8])
	cacheZip := filepath.Join(dl.cacheDir, filepath.Dir(sourceName), filename)
	// If cached ZIP does not exist - fetch the archive
	if _, err := os.Stat(cacheZip); err != nil {
		if err = os.MkdirAll(filepath.Join(dl.cacheDir, filepath.Dir(sourceName)), 0755); err != nil {
			return "", err
		}
		// TODO: Ref discovery
		slog.Info("Cache miss. Loading", slog.String("source", source))
		if err = gitArchive(source, ref, cacheZip); err != nil {
			return "", err
		}
	} else {
		slog.Info("Cache hit. Loading cache.", slog.String("path", filename))
	}
	return cacheZip, nil
}

func (dl *goLikeDownloader) discoverSource(source string) ([]byte, error) {
	// TODO: Better dependency path handling
	// Reuse the same resolution mechanism that go mod uses
	// https://go.dev/ref/mod#vcs-find
	url, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	query := url.Query()
	query.Add("go-get", "1")

	resp, err := http.Get(url.String() + "?" + query.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (dl *goLikeDownloader) parseGoQuery(goQuery string) (string, string, string) {
	parts := strings.Split(goQuery, " ")
	return parts[0], parts[1], parts[2]
}
