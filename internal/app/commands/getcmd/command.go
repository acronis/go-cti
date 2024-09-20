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
	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/depman"
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

func (c *cmd) Execute(_ context.Context) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	idxFile := filepath.Join(workDir, bundle.IndexFileName)

	slog.Info("Loading bundle", slog.String("index", idxFile))
	p, err := depman.New(idxFile)
	if err != nil {
		return fmt.Errorf("create package manager: %w", err)
	}

	var deps []string
	if len(c.targets) != 0 {
		for i := range c.targets {
			// TODO: Potentially unreliable
			c.targets[i] = strings.ReplaceAll(c.targets[i], "@", " ")
		}
		installed, err := p.InstallNewDependencies(c.targets, c.getOpts.Replace)
		if err != nil {
			return fmt.Errorf("install dependencies: %w", err)
		}
		deps = installed
	} else {
		installed, err := p.InstallIndexDependencies()
		if err != nil {
			return fmt.Errorf("install dependencies: %w", err)
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
