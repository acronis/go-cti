package packcmd

import (
	"context"
	"fmt"
	"log/slog"

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
	bd, err := func() (*bundle.Bundle, error) {
		if len(c.targets) == 0 {
			slog.Info("Packing metadata for the current bundle...")
			return bundle.New("")
		}
		slog.Info("Packing metadata", slog.String("target", c.targets[0]))
		return bundle.New(c.targets[0])
	}()
	if err != nil {
		return fmt.Errorf("initialize a new bundle: %w", err)
	}

	filename, err := bunman.Pack(bd, c.packOpts.IncludeSource)
	if err != nil {
		return fmt.Errorf("pack the bundle: %w", err)
	}

	slog.Info("Packing has been completed", "filename", filename)

	return nil
}
