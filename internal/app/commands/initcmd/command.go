package initcmd

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
		Use:   "init",
		Short: "generate cti project with default dependencies",
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
	slog.Info("Initialize package", slog.String("path", baseDir))
	pkg := ctipackage.New(baseDir)
	if pkg.Read() == nil {
		slog.Info("Package already initialized")
		return nil
	}

	if err := pkg.Initialize(); err != nil {
		return fmt.Errorf("initialize the package: %w", err)
	}

	slog.Info("Package was initialized")
	return nil
}
