package downloadcmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/pkg/ctipackage"
	"github.com/acronis/go-cti/pkg/pacman"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "download",
		Short: "command to download CyberApp(s) from a remote repository into the cache",
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

				return command.WrapError(downloadPackages(ctx, pm, packages))
			}

			return command.WrapError(downloadPackageDependencies(ctx, pm, baseDir))
		},
	}
}

func downloadPackages(_ context.Context, pm pacman.PackageManager, packages map[string]string) error {
	slog.Info("Download",
		slog.Any("packages", packages),
	)

	if _, err := pm.Download(packages); err != nil {
		return fmt.Errorf("download packages: %w", err)
	}

	// TODO probably show information about downloaded packages
	slog.Info("Packages were downloaded")
	return nil
}

func downloadPackageDependencies(_ context.Context, pm pacman.PackageManager, baseDir string) error {
	slog.Info("Download package dependencies",
		slog.String("path", baseDir),
	)

	pkg, err := ctipackage.New(baseDir)
	if err != nil {
		return fmt.Errorf("new package: %w", err)
	}
	if err := pkg.Read(); err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	if _, err := pm.Download(pkg.Index.Depends); err != nil {
		return fmt.Errorf("download packages: %w", err)
	}

	// TODO probably show information about downloaded packages
	slog.Info("Packages were downloaded")
	return nil
}
