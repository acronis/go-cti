package depman

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/downloader"
	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/parser"
)

const (
	DependencyDirName = ".dep"
)

type DependencyManager interface {
	InstallNewDependencies(depends []string, replace bool) ([]string, error)
	InstallIndexDependencies() ([]string, error)
}

type dependencyManager struct {
	RootBundle      *bundle.Bundle
	BundlesCacheDir string
	DependenciesDir string
	Downloader      downloader.Downloader

	BaseDir string
}

func New(bd *bundle.Bundle) (DependencyManager, error) {
	cacheDir, err := filesys.GetCtiBundlesCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get cache dir: %w", err)
	}
	dependsDir := filepath.Join(bd.BaseDir, DependencyDirName)

	return &dependencyManager{
		RootBundle:      bd,
		BundlesCacheDir: cacheDir,
		DependenciesDir: dependsDir,
		BaseDir:         bd.BaseDir,
		Downloader:      downloader.New(bd.IndexLock, cacheDir, dependsDir),
	}, nil
}

func NewWithDownloader(bd *bundle.Bundle, dl downloader.Downloader) (DependencyManager, error) {
	cacheDir, err := filesys.GetCtiBundlesCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get cache dir: %w", err)
	}

	return &dependencyManager{
		RootBundle:      bd,
		BundlesCacheDir: cacheDir,
		DependenciesDir: filepath.Join(bd.BaseDir, DependencyDirName),
		BaseDir:         bd.BaseDir,
		Downloader:      dl,
	}, nil
}

func (dm *dependencyManager) InstallNewDependencies(depends []string, replace bool) ([]string, error) {
	installed, replaced, err := dm.installDependencies(depends, replace)
	if err != nil {
		return nil, fmt.Errorf("install dependencies: %w", err)
	}

	// TODO: Possibly needs refactor
	if len(replaced) != 0 {
		var depends []string
		for _, idxDepName := range dm.RootBundle.Index.Depends {
			depName, _ := bundle.ParseIndexDependency(idxDepName)
			if _, ok := replaced[depName]; ok {
				continue
			}
			depends = append(depends, idxDepName)
		}
		dm.RootBundle.Index.Depends = depends
	}

	for _, depName := range depends {
		found := false
		for _, idxDepName := range dm.RootBundle.Index.Depends {
			if idxDepName == depName {
				found = true
				break
			}
		}
		if !found {
			dm.RootBundle.Index.Depends = append(dm.RootBundle.Index.Depends, depName)
			slog.Info("Added direct dependency", slog.String("bundle", depName))
		}
	}

	if err = dm.RootBundle.SaveIndex(); err != nil {
		return nil, fmt.Errorf("save index: %w", err)
	}

	if err = dm.RootBundle.SaveIndexLock(); err != nil {
		return nil, fmt.Errorf("save index lock: %w", err)
	}

	return installed, nil
}

func (dm *dependencyManager) InstallIndexDependencies() ([]string, error) {
	installed, _, err := dm.installDependencies(dm.RootBundle.Index.Depends, false)
	if err != nil {
		return nil, fmt.Errorf("install index dependencies: %w", err)
	}
	if err = dm.RootBundle.SaveIndexLock(); err != nil {
		return nil, fmt.Errorf("save index lock: %w", err)
	}
	return installed, nil
}

func (dm *dependencyManager) installDependencies(depends []string, replace bool) ([]string, map[string]struct{}, error) {
	installed, replaced, err := dm.Downloader.Download(depends, replace)
	if err != nil {
		return nil, nil, fmt.Errorf("download dependencies: %w", err)
	}
	if err = dm.processInstalledDependencies(installed); err != nil {
		return nil, nil, fmt.Errorf("process installed dependencies: %w", err)
	}
	return installed, replaced, nil
}

func (dm *dependencyManager) processInstalledDependencies(installed []string) error {
	for _, sourceName := range installed {
		pkgLock := dm.RootBundle.IndexLock.Bundles[sourceName]
		pkgPath := filepath.Join(dm.DependenciesDir, pkgLock.AppCode)
		for _, dep := range pkgLock.Depends {
			depSourceName, _ := bundle.ParseIndexDependency(dep)
			depBundleLock := dm.RootBundle.IndexLock.Bundles[depSourceName]

			if err := dm.rewriteDepLinks(pkgPath, depBundleLock.AppCode); err != nil {
				return fmt.Errorf("rewrite dependency links: %w", err)
			}
		}

		bd, err := bundle.New(pkgPath)
		if err != nil {
			return fmt.Errorf("new bundle: %w", err)
		}
		if err := parser.BuildPackageCache(bd); err != nil {
			return fmt.Errorf("build cache: %w", err)
		}
	}
	return nil
}

func (dm *dependencyManager) rewriteDepLinks(pkgPath, depName string) error {
	relPath, err := filepath.Rel(pkgPath, dm.RootBundle.BaseDir)
	if err != nil {
		return err
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	orig := fmt.Sprintf("%s/%s", DependencyDirName, depName)
	repl := fmt.Sprintf("%s/%s/%s", relPath, DependencyDirName, depName)

	for _, file := range filesys.WalkDir(pkgPath, ".raml") {
		// TODO: Maybe read file line by line?
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		contents := strings.ReplaceAll(string(raw), orig, repl)
		err = os.WriteFile(file, []byte(contents), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}
