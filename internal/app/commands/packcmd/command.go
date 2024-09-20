package packcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/depman"
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

func (c *cmd) Execute(_ context.Context) error {
	var workDir string
	if len(c.targets) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		workDir = wd
		slog.Info("Packing metadata for the current bundle...")
	} else {
		workDir = filepath.Join(depman.DependencyDirName, c.targets[0])
		slog.Info(fmt.Sprintf("Packing metadata for %s...", c.targets[0]))
	}
	idxFile := filepath.Join(workDir, bundle.IndexFileName)

	pm, err := depman.New(idxFile)
	if err != nil {
		return fmt.Errorf("create package manager: %w", err)
	}
	filename, err := pm.Pack(c.packOpts.IncludeSource)
	if err != nil {
		return fmt.Errorf("pack the bundle: %w", err)
	}

	slog.Info("Packing has been completed", "filename", filename)

	return nil
}
