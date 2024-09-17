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
	"github.com/acronis/go-cti/pkg/parser"
)

type cmd struct {
	opts    cti.Options
	targets []string
}

func New(opts cti.Options, targets []string) command.Command {
	return &cmd{
		opts:    opts,
		targets: targets,
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
		slog.Info("Building metadata for the current package...")
	} else {
		workDir = filepath.Join(pacman.DependencyDirName, c.targets[0])
		slog.Info(fmt.Sprintf("Building metadata for %s...", c.targets[0]))
	}
	idxFile := filepath.Join(workDir, _package.IndexFileName)

	p, err := parser.ParsePackage(idxFile)
	if err != nil {
		return err
	}
	if err = p.DumpCache(); err != nil {
		return err
	}
	slog.Info("Done!")

	return nil
}
