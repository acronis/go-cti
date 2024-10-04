package synccmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/pkg/bundle"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "synchronize bundle directory content with the index",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := command.GetWorkingDir(cmd)
			if err != nil {
				return fmt.Errorf("get base directory: %w", err)
			}

			return command.WrapError(execute(ctx, baseDir))
		},
	}
}

func execute(_ context.Context, baseDir string) error {
	slog.Info("Synchronize bundle directory", slog.String("path", baseDir))
	bd := bundle.New(baseDir)
	if bd.Read() != nil {
		slog.Info("Failed to read bundle, you can reinitialize it with 'cti init' command")
		return nil
	}

	if err := bd.Sync(); err != nil {
		return fmt.Errorf("sync bundle: %w", err)
	}

	slog.Info("Bundle directory was synchronized")
	return nil
}
