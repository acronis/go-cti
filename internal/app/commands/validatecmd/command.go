package validatecmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/bunman"
	"github.com/acronis/go-stacktrace"
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

	slog.Info("Validating bundle", slog.String("path", bundlePath))
	bd := bundle.New(bundlePath)
	if err := bd.Read(); err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	// TODO: Validation for usage of indirect dependencies
	if errs := bunman.Validate(bd); errs != nil {
		for i := range errs {
			slog.Error("Validation error", stacktrace.ErrToSlogAttr(errs[i]))
		}
		return errors.New("failed to validate the bundle")
	}
	slog.Info("No errors found")
	return nil
}
