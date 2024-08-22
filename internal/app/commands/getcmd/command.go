package getcmd

import (
	"context"
	"encoding/json"
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

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	"github.com/acronis/go-cti/pkg/depman"
	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/index"
	"github.com/acronis/go-cti/pkg/parser"
)

var goImportRe = regexp.MustCompile("<meta name=\"go-import\" content=\"([^\"]+)")

type cmd struct {
	opts    cti.Options
	getOpts GetOptions
	targets []string
}

type GetOptions struct {
	Replace bool
}

func New(opts cti.Options, getOpts GetOptions, targets []string) command.Command {
	return &cmd{
		opts:    opts,
		getOpts: getOpts,
		targets: targets,
	}
}

func buildCache(workDir string) error {
	parser, err := parser.NewRamlParser(filepath.Join(workDir, "index.json"))
	if err != nil {
		return err
	}
	if err = parser.Bundle(workDir); err != nil {
		return err
	}
	return nil
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
	re := regexp.MustCompile(`\s+`)
	refData := strings.Split(re.ReplaceAllString(string(out), " "), " ")
	return refData[0], nil
}

func loadGitDependency(pkgCacheDir string, sourceName string, source string, ref string, hash string) (string, error) {
	filename := fmt.Sprintf("%s-%s-%s.zip", filepath.Base(sourceName), ref, hash[:8])
	cacheZip := filepath.Join(pkgCacheDir, filepath.Dir(sourceName), filename)
	// If cached ZIP does not exist - fetch the archive
	if _, err := os.Stat(cacheZip); err != nil {
		if err = os.MkdirAll(filepath.Join(pkgCacheDir, filepath.Dir(sourceName)), 0755); err != nil {
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

func loadSourceInfo(source string) ([]byte, error) {
	// TODO: Better dependency path handling
	// Reuse the same resolution mechanism that go mod uses
	// https://go.dev/ref/mod#vcs-find
	// TODO: Generalize loader and dependency management code
	url, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	query := url.Query()
	query.Add("go-get", "1")

	return func() ([]byte, error) {
		resp, err := http.Get(url.String() + "?" + query.Encode())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		return io.ReadAll(resp.Body)
	}()
}

func parseGoQuery(goQuery string) (string, string, string) {
	parts := strings.Split(goQuery, " ")
	return parts[0], parts[1], parts[2]
}

func rewriteDepLinks(workDir string, location string, depName string) error {
	relPath, err := filepath.Rel(location, workDir)
	if err != nil {
		return err
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	orig := fmt.Sprintf(".dep/%s", depName)
	repl := fmt.Sprintf("%s/.dep/%s", relPath, depName)

	for _, file := range filesys.WalkDir(location, ".raml") {
		// TODO: Maybe read file line by line?
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		contents := strings.ReplaceAll(string(raw), orig, repl)
		err = os.WriteFile(file, []byte(contents), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func download(workDir string, pkgCacheDir string, depends []string, idxLock *depman.IndexLock, replace bool) ([]string, map[string]struct{}, error) {
	var replaced = make(map[string]struct{})
	var installed []string
	for _, dep := range depends {
		// TODO: Dependency consists of two space-delimited parts:
		// 1. Dependency name
		// 2. Dependency version
		depInfo := strings.Split(dep, " ")
		sourceName := depInfo[0]
		depVersion := depInfo[1]

		source := fmt.Sprintf("https://%s", sourceName)
		body, err := loadSourceInfo(source)
		if err != nil {
			return nil, nil, err
		}

		m := goImportRe.FindStringSubmatch(string(body))
		if len(m) == 0 {
			return nil, nil, fmt.Errorf("failed to find go-import at %s", source)
		}
		slog.Info(fmt.Sprintf("Discovered dependency %s", sourceName))
		_, _, sourceLocation := parseGoQuery(m[len(m)-1])

		// FIXME: This will only work with git source!
		commitHash, err := gitLsRemote(sourceLocation, depVersion)
		if err != nil {
			return nil, nil, err
		} else if commitHash == "" {
			return nil, nil, fmt.Errorf("failed to find %s %s", sourceName, depVersion)
		}

		if pkg, ok := idxLock.Packages[sourceName]; ok {
			// TODO: Package version comparison using semver?
			if pkg.Integrity == commitHash {
				slog.Info("Package did not change. Skipping.")
				continue
			}
		}

		// go-import consists of space-delimited data with:
		// 1. Dependency name
		// 2. Source type (mod, vcs, git)
		// 3. Source location
		// TODO: Support other source types?
		cacheZip, err := loadGitDependency(pkgCacheDir, sourceName, sourceLocation, depVersion, commitHash)
		if err != nil {
			return nil, nil, err
		}

		data, err := filesys.OpenZipFile(cacheZip, "index.json")
		if err != nil {
			return nil, nil, err
		}
		depIdx, err := index.DecodeIndexBytes(data)
		if err != nil {
			return nil, nil, err
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
		if s, ok := idxLock.Sources[depIdx.AppCode]; ok && s.Source != sourceName {
			slog.Warn(fmt.Sprintf("%s was already installed from %s.", depIdx.AppCode, s.Source))
			if !replace {
				continue
			}
			slog.Warn(fmt.Sprintf("Replacing %s with %s.", s.Source, sourceName))
			delete(idxLock.Packages, s.Source)
			replaced[s.Source] = struct{}{}
		}

		dest := filepath.Join(workDir, ".dep", depIdx.AppCode)
		if _, err := os.Stat(dest); err == nil {
			if err = os.RemoveAll(dest); err != nil {
				return nil, nil, err
			}
		}

		if _, err = filesys.UnzipToFS(cacheZip, dest); err != nil {
			return nil, nil, err
		}

		idxLock.Sources[depIdx.AppCode] = depman.SourceInfo{
			Source: sourceName,
		}

		idxLock.Packages[sourceName] = depman.PackageInfo{
			Name:      "",
			AppCode:   depIdx.AppCode,
			Integrity: commitHash,
			Version:   depVersion, // TODO: Use golang pseudo-version format
			Source:    source,
			Depends:   depIdx.Depends,
		}

		if depIdx.Depends != nil {
			depInstalled, depReplaced, err := download(workDir, pkgCacheDir, depIdx.Depends, idxLock, replace)
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

func (c *cmd) Execute(ctx context.Context) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	idxFile := filepath.Join(workDir, "index.json")

	pkgCacheDir, err := filesys.GetPkgCacheDir()
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Loading index file %s", idxFile))
	idx, err := index.OpenIndexFile(idxFile)
	if err != nil {
		return err
	}

	idxLockPath := filepath.Join(workDir, "index-lock.json")
	idxLock := depman.MakeIndexLock()
	if data, err := os.ReadFile(idxLockPath); err == nil {
		if err = json.Unmarshal(data, idxLock); err != nil {
			return err
		}
	}

	// TODO: Refactor
	if len(c.targets) != 0 {
		// TODO: Work without specific version
		depends := make([]string, 0, len(c.targets))
		for _, depName := range c.targets {
			targetInfo := strings.Split(depName, "@")
			targetName := targetInfo[0]
			targetVersion := targetInfo[1]
			depends = append(depends, fmt.Sprintf("%s %s", targetName, targetVersion))
		}

		// TODO: Most likely need to make it an outer for-loop for more control
		installed, replaced, err := download(workDir, pkgCacheDir, depends, idxLock, c.getOpts.Replace)
		if err != nil {
			return err
		}

		for _, sourceName := range installed {
			pkgLock := idxLock.Packages[sourceName]
			pkgPath := filepath.Join(workDir, ".dep", pkgLock.AppCode)
			for _, dep := range pkgLock.Depends {
				depInfo := strings.Split(dep, " ")
				depSourceName := depInfo[0]
				depPkgLock := idxLock.Packages[depSourceName]
				err := rewriteDepLinks(workDir, pkgPath, depPkgLock.AppCode)
				if err != nil {
					return err
				}
			}
		}

		for _, sourceName := range installed {
			pkgLock := idxLock.Packages[sourceName]
			if err := buildCache(filepath.Join(workDir, ".dep", pkgLock.AppCode)); err != nil {
				return err
			}
		}

		if installed != nil {
			data, _ := json.MarshalIndent(idxLock, "", "  ")
			if err = os.WriteFile(idxLockPath, data, 0755); err != nil {
				return err
			}

			slog.Info(fmt.Sprintf("Installed: %s", strings.Join(installed, ", ")))
		} else {
			slog.Info("Nothing to install.")
		}

		// TODO: Possibly needs refactor
		if len(replaced) != 0 {
			var depends []string
			for _, idxDepName := range idx.Data.Depends {
				depInfo := strings.Split(idxDepName, " ")
				depName := depInfo[0]
				if _, ok := replaced[depName]; ok {
					continue
				}
				depends = append(depends, idxDepName)
			}
			idx.Data.Depends = depends
			if err = idx.Save(); err != nil {
				return err
			}
		}

		for _, depName := range depends {
			found := false
			for _, idxDepName := range idx.Data.Depends {
				if idxDepName == depName {
					found = true
					break
				}
			}
			if !found {
				idx.Data.Depends = append(idx.Data.Depends, depName)
				if err = idx.Save(); err != nil {
					return err
				}
				slog.Info(fmt.Sprintf("Added %s as direct dependency", c.targets[0]))
			}
		}
	} else {
		installed, _, err := download(workDir, pkgCacheDir, idx.Data.Depends, idxLock, c.getOpts.Replace)
		if err != nil {
			return err
		}

		for _, sourceName := range installed {
			pkgLock := idxLock.Packages[sourceName]
			pkgPath := filepath.Join(workDir, ".dep", pkgLock.AppCode)
			for _, dep := range pkgLock.Depends {
				depInfo := strings.Split(dep, " ")
				depSourceName := depInfo[0]
				depPkgLock := idxLock.Packages[depSourceName]
				err := rewriteDepLinks(workDir, pkgPath, depPkgLock.AppCode)
				if err != nil {
					return err
				}
			}
		}

		for _, sourceName := range installed {
			pkgLock := idxLock.Packages[sourceName]
			if err := buildCache(filepath.Join(workDir, ".dep", pkgLock.AppCode)); err != nil {
				return err
			}
		}

		if installed != nil {
			data, _ := json.MarshalIndent(idxLock, "", "  ")
			if err = os.WriteFile(idxLockPath, data, 0755); err != nil {
				return err
			}

			slog.Info(fmt.Sprintf("Installed: %s", strings.Join(installed, ", ")))
		} else {
			slog.Info("Nothing to install.")
		}
	}

	return nil
}
