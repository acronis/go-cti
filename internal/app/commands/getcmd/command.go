package getcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	_package "github.com/acronis/go-cti/pkg/package"
	"github.com/acronis/go-cti/pkg/pacman"
)

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

func (c *cmd) Execute(ctx context.Context) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	idxFile := filepath.Join(workDir, _package.IndexFileName)

	slog.Info(fmt.Sprintf("Loading package at %s", idxFile))
	p, err := pacman.New(idxFile)
	if err != nil {
		return fmt.Errorf("failed to create package manager: %w", err)
	}

	var deps []string
	if len(c.targets) != 0 {
		for i := range c.targets {
			// TODO: Potentially unreliable
			c.targets[i] = strings.Replace(c.targets[i], "@", " ", -1)
		}
		installed, err := p.InstallNewDependencies(c.targets, c.getOpts.Replace)
		if err != nil {
			return fmt.Errorf("failed to install dependencies: %w", err)
		}
		deps = installed
	} else {
		installed, err := p.InstallIndexDependencies()
		if err != nil {
			return fmt.Errorf("failed to install dependencies: %w", err)
		}
		deps = installed
	}
	if deps != nil {
		slog.Info(fmt.Sprintf("Installed: %s", strings.Join(deps, ", ")))
	} else {
		slog.Info("Nothing to install.")
	}

	return nil
}
