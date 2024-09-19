package packcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	_package "github.com/acronis/go-cti/pkg/package"
	"github.com/acronis/go-cti/pkg/pacman"
)

type cmd struct {
	opts     cti.Options
	packOpts PackOptions
	targets  []string
}

type PackOptions struct {
	IncludeSource bool
}

func New(opts cti.Options, packOpts PackOptions, targets []string) command.Command {
	return &cmd{
		opts:     opts,
		packOpts: packOpts,
		targets:  targets,
	}
}

func (c *cmd) Execute(ctx context.Context) error {
	var workDir string
	if len(c.targets) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		workDir = wd
		slog.Info("Packing metadata for the current package...")
	} else {
		workDir = filepath.Join(pacman.DependencyDirName, c.targets[0])
		slog.Info(fmt.Sprintf("Packing metadata for %s...", c.targets[0]))
	}
	idxFile := filepath.Join(workDir, _package.IndexFileName)

	pm, err := pacman.New(idxFile)
	if err != nil {
		return fmt.Errorf("failed to create package manager: %w", err)
	}
	if err := pm.Pack(c.packOpts.IncludeSource); err != nil {
		return fmt.Errorf("failed to pack the package: %w", err)
	}

	slog.Info("Packing has been completed", "filename", filepath.Join(pm.BaseDir, pacman.BundleName))

	return nil
}
