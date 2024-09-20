package pacman

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/acronis/go-cti/pkg/filesys"
	_package "github.com/acronis/go-cti/pkg/package"
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

type GoLikeDownloader struct {
	indexLock       *_package.IndexLock
	packageCacheDir string
	dependenciesDir string
}

func NewGoLikeDownloader(pacman *PackageManager) *GoLikeDownloader {
	return &GoLikeDownloader{
		indexLock:       pacman.Package.IndexLock,
		packageCacheDir: pacman.PackageCacheDir,
		dependenciesDir: pacman.DependenciesDir,
	}
}

func (dl *GoLikeDownloader) loadGitDependency(sourceName string, source string, ref string, hash string) (string, error) {
	filename := fmt.Sprintf("%s-%s-%s.zip", filepath.Base(sourceName), ref, hash[:8])
	cacheZip := filepath.Join(dl.packageCacheDir, filepath.Dir(sourceName), filename)
	// If cached ZIP does not exist - fetch the archive
	if _, err := os.Stat(cacheZip); err != nil {
		if err = os.MkdirAll(filepath.Join(dl.packageCacheDir, filepath.Dir(sourceName)), 0755); err != nil {
			return "", err
		}
		// TODO: Ref discovery
		slog.Info(fmt.Sprintf("Cache miss. Loading from: %s", source))
		if err = gitArchive(source, ref, cacheZip); err != nil {
			return "", err
		}
	} else {
		slog.Info(fmt.Sprintf("Cache hit. Loading %s from cache.", filename))
	}
	return cacheZip, nil
}

func (dl *GoLikeDownloader) Download(depends []string, replace bool) ([]string, map[string]struct{}, error) {
	var replaced = make(map[string]struct{})
	var installed []string
	for _, dep := range depends {
		// TODO: Dependency consists of two space-delimited parts:
		// 1. Dependency name
		// 2. Dependency version
		sourceName, depVersion := ParseIndexDependency(dep)
		if depVersion == "" {
			depVersion = "main"
		}

		source := fmt.Sprintf("https://%s", sourceName)
		body, err := dl.discoverSource(source)
		if err != nil {
			return nil, nil, err
		}

		m := goImportRe.FindStringSubmatch(string(body))
		if len(m) == 0 {
			return nil, nil, fmt.Errorf("failed to find go-import at %s", source)
		}
		slog.Info(fmt.Sprintf("Discovered dependency %s", sourceName))
		_, _, sourceLocation := dl.parseGoQuery(m[len(m)-1])

		// FIXME: This will only work with git source!
		commitHash, err := gitLsRemote(sourceLocation, depVersion)
		if err != nil {
			return nil, nil, err
		} else if commitHash == "" {
			return nil, nil, fmt.Errorf("failed to find %s %s", sourceName, depVersion)
		}

		if pkg, ok := dl.indexLock.Packages[sourceName]; ok {
			// TODO: Package version comparison using semver?
			_, err := os.Stat(filepath.Join(dl.dependenciesDir, pkg.AppCode))
			if pkg.Integrity == commitHash && err == nil {
				slog.Info("Package did not change. Skipping.")
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

		rc, err := filesys.OpenZipFile(cacheZip, _package.IndexFileName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open index.json in %s: %w", cacheZip, err)
		}
		depIdx, err := _package.UnmarshalIndexFile(rc)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal index.json in %s: %w", cacheZip, err)
		}

		if depIdx.AppCode == "" {
			return nil, nil, fmt.Errorf("package at %s contains empty application code", sourceName)
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
			delete(dl.indexLock.Packages, s.Source)
			replaced[s.Source] = struct{}{}
		}

		dest := filepath.Join(dl.dependenciesDir, depIdx.AppCode)
		if _, err := os.Stat(dest); err == nil {
			if err = os.RemoveAll(dest); err != nil {
				return nil, nil, err
			}
		}

		if _, err = filesys.UnzipToFS(cacheZip, dest); err != nil {
			return nil, nil, err
		}

		dl.indexLock.Sources[depIdx.AppCode] = _package.SourceInfo{
			Source: sourceName,
		}

		dl.indexLock.Packages[sourceName] = _package.PackageInfo{
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

func (dl *GoLikeDownloader) discoverSource(source string) ([]byte, error) {
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

func (dl *GoLikeDownloader) parseGoQuery(goQuery string) (string, string, string) {
	parts := strings.Split(goQuery, " ")
	return parts[0], parts[1], parts[2]
}

// TODO: Maybe use go-git. But it doesn't have git archive...
func gitArchive(remote string, ref string, destination string) error {
	cmd := exec.Command("git", "archive", "--remote", remote, ref, "-o", destination)
	slog.Info(fmt.Sprintf("Executing command: %s", cmd.String()))
	if _, err := cmd.Output(); err != nil {
		return err
	}
	return nil
}

func gitLsRemote(remote string, ref string) (string, error) {
	cmd := exec.Command("git", "ls-remote", remote, ref)
	slog.Info(fmt.Sprintf("Executing command: %s", cmd.String()))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	refData := strings.Split(wsRe.ReplaceAllString(string(out), " "), " ")
	return refData[0], nil
}
