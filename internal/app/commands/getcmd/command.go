package getcmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/depman"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "tool to download cti bundles from a remote repository",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := command.GetWorkingDir(cmd)
			if err != nil {
				return fmt.Errorf("get base directory: %w", err)
			}

			return command.WrapError(execute(ctx, baseDir, args))
		},
	}
}

func execute(_ context.Context, baseDir string, targets []string) error {
	slog.Info("Get depends for bundle",
		slog.String("path", baseDir),
		slog.Any("targets", targets),
	)

	bd := bundle.New(baseDir)
	if err := bd.Read(); err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	dm, err := depman.New()
	if err != nil {
		return fmt.Errorf("create package manager: %w", err)
	}

	if len(targets) != 0 {
		depends := make(map[string]string)
		for _, tgt := range targets {
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
