package packcmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/bunman"
	"github.com/spf13/cobra"
)

type PackOptions struct {
	IncludeSource bool
}

func New(ctx context.Context) *cobra.Command {
	packOpts := PackOptions{}
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "pack cti bundle",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := command.GetWorkingDir(cmd)
			if err != nil {
				return fmt.Errorf("get base directory: %w", err)
			}

			return command.WrapError(execute(ctx, baseDir, packOpts))
		},
	}

	cmd.Flags().BoolVarP(&packOpts.IncludeSource, "include-source", "s", false, "Include source files in the resulting bundle.")
	return cmd
}

func execute(_ context.Context, baseDir string, opts PackOptions) error {
	slog.Info("Packing bundle", slog.String("path", baseDir))
	bd := bundle.New(baseDir)
	if err := bd.Read(); err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	filename, err := bunman.Pack(bd, opts.IncludeSource)
	if err != nil {
		return fmt.Errorf("pack the bundle: %w", err)
	}

	slog.Info("Packing has been completed", "filename", filename)

	return nil
}
