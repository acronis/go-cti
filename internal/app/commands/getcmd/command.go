package getcmd

import (
	"context"
	"fmt"
	"log/slog"
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
	bd, err := bundle.New("")
	if err != nil {
		return fmt.Errorf("initialize a new bundle: %w", err)
	}

	dm, err := depman.New(bd)
	if err != nil {
		return fmt.Errorf("create package manager: %w", err)
	}

	var deps []string
	if len(c.targets) != 0 {
		for i := range c.targets {
			// TODO: Potentially unreliable
			c.targets[i] = strings.ReplaceAll(c.targets[i], "@", " ")
		}
		installed, err := dm.InstallNewDependencies(c.targets, c.getOpts.Replace)
		if err != nil {
			return fmt.Errorf("install dependencies: %w", err)
		}
		deps = installed
	} else {
		installed, err := dm.InstallIndexDependencies()
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
