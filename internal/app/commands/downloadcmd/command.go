package downloadcmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/pkg/depman"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "download",
		Short: "command to download cti package(s) from a remote repository into the cache",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.WrapError(execute(ctx, args))
		},
	}
}

func execute(_ context.Context, packages []string) error {
	slog.Info("Download",
		slog.Any("packages", packages),
	)

	pkgs := map[string]string{}
	for _, pkg := range packages {
		chunks := strings.Split(pkg, "@")
		if len(chunks) != 2 {
			return fmt.Errorf("invalid package format: %s, should be `<source>@<version>`", pkg)
		}
		if _, ok := pkgs[chunks[0]]; ok {
			return fmt.Errorf("duplicate package: %s", chunks[0])
		}
		pkgs[chunks[0]] = chunks[1]
	}

	dm, err := depman.New()
	if err != nil {
		return fmt.Errorf("create package manager: %w", err)
	}

	if _, err := dm.Download(pkgs); err != nil {
		return fmt.Errorf("download packages: %w", err)
	}

	// TODO probably show information about downloaded packages
	slog.Info("Packages were downloaded")
	return nil
}
