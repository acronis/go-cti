package getcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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
}

func New(opts cti.Options, getOpts GetOptions, targets []string) command.Command {
	return &cmd{
		opts:    opts,
		getOpts: getOpts,
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

	slog.Info("Get depends for bundle", slog.String("path", bundlePath))

	bd := bundle.New(bundlePath)
	if err := bd.Read(); err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	dm, err := depman.New()
	if err != nil {
		return fmt.Errorf("create package manager: %w", err)
	}

	if len(c.targets) != 0 {
		depends := make(map[string]string)
		for _, tgt := range c.targets {
			chunks := strings.Split(tgt, "@")
			if len(chunks) != 2 {
				return fmt.Errorf("invalid depends format: %s, should be `<source>@<version>`", tgt)
			}
			if _, ok := depends[chunks[0]]; ok {
				return fmt.Errorf("duplicate dependency: %s", chunks[0])
			}
			depends[chunks[0]] = chunks[1]
		}

		if err := dm.Add(bd, depends); err != nil {
			return fmt.Errorf("install dependencies: %w", err)
		}
	} else {
		if err := dm.Install(bd); err != nil {
			return fmt.Errorf("install dependencies: %w", err)
		}
	}

	return nil
}
