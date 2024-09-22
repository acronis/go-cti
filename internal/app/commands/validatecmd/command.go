package validatecmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/bunman"
	"github.com/acronis/go-cti/pkg/slogex"
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

func (c *cmd) Execute(_ context.Context) error {
	bd, err := func() (*bundle.Bundle, error) {
		if len(c.targets) == 0 {
			slog.Info("Validating current bundle...")
			return bundle.New("")
		}
		slog.Info(fmt.Sprintf("Validating bundle in %s...", c.targets[0]))
		return bundle.New(c.targets[0])
	}()
	if err != nil {
		return fmt.Errorf("initialize a new bundle: %w", err)
	}

	// TODO: Validation for usage of indirect dependencies
	if errs := bunman.Validate(bd); errs != nil {
		for i := range errs {
			slog.Error("Validation error", slogex.ErrorWithTrace(errs[i]))
		}
		return errors.New("failed to validate the bundle")
	}
	slog.Info("No errors found")
	return nil
}
