package getcmd

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
		Use:   "pkg",
		Short: "command to add new or install cti from cache",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := command.GetWorkingDir(cmd)
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			pm, err := command.InitializePackageManager(cmd)
			if err != nil {
				return fmt.Errorf("initialize package manager: %w", err)
			}

			if len(args) > 0 {
				packages, err := command.ParsePackages(args)
				if err != nil {
					return fmt.Errorf("parse packages: %w", err)
				}

				return command.WrapError(addPackages(ctx, baseDir, pm, packages))
			}

			return command.WrapError(installAll(ctx, baseDir, pm))
		},
	}
}

func addPackages(_ context.Context, baseDir string, pm pacman.PackageManager, packages map[string]string) error {
	slog.Info("Add package dependencies",
		slog.String("path", baseDir),
		slog.Any("packages", packages),
	)

	pkg, err := ctipackage.New(baseDir)
	if err != nil {
		return fmt.Errorf("new package: %w", err)
	}
	if err := pkg.Read(); err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	if err := pm.Add(pkg, packages); err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	return nil
}

func installAll(_ context.Context, baseDir string, pm pacman.PackageManager) error {
	slog.Info("Install dependent packages",
		slog.String("path", baseDir),
	)

	pkg, err := ctipackage.New(baseDir)
	if err != nil {
		return fmt.Errorf("new package: %w", err)
	}
	if err := pkg.Read(); err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	if err := pm.Install(pkg, false); err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	return nil
}
