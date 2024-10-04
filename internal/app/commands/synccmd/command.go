package synccmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/pkg/ctipackage"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "synchronize package directory content with the index",
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
	slog.Info("Synchronize package directory", slog.String("path", baseDir))
	pkg := ctipackage.New(baseDir)
	if pkg.Read() != nil {
		slog.Info("Failed to read package, you can reinitialize it with 'cti init' command")
		return nil
	}

	if err := pkg.Sync(); err != nil {
		return fmt.Errorf("sync package: %w", err)
	}

	slog.Info("Package directory was synchronized")
	return nil
}
