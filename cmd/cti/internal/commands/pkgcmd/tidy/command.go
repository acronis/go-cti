package tidycmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/cmd/cti/internal/command"

	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/pacman"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "tidy",
		Short: "command to sync package directory with content and index/index",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			baseDir, err := command.GetWorkingDir(cmd)
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			pm, err := command.InitializePackageManager(cmd)
			if err != nil {
				return fmt.Errorf("initialize package manager: %w", err)
			}

			return command.WrapError(tidy(ctx, baseDir, pm))
		},
	}
}

func tidy(_ context.Context, baseDir string, pm pacman.PackageManager) error {
	slog.Info("Install all packages",
		slog.String("path", baseDir),
	)

	pkg, err := ctipackage.New(baseDir)
	if err != nil {
		return fmt.Errorf("new package: %w", err)
	}
	if err := pkg.Read(); err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	if err := pm.Install(pkg, true); err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	return nil
}
