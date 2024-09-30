package packcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/bunman"
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
	bundlePath, err := func() (string, error) {
		if len(c.targets) == 0 {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("get current working directory: %w", err)
			}
			return cwd, nil
		}
		return c.targets[0], nil
	}()
	if err != nil {
		return fmt.Errorf("get bundle path: %w", err)
	}

	slog.Info("Packing bundle", slog.String("path", bundlePath))
	bd := bundle.New(bundlePath)
	if err := bd.Read(); err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	filename, err := bunman.Pack(bd, c.packOpts.IncludeSource)
	if err != nil {
		return fmt.Errorf("pack the bundle: %w", err)
	}

	slog.Info("Packing has been completed", "filename", filename)

	return nil
}
