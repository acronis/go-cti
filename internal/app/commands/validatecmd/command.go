package validatecmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/internal/app/command"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/bunman"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "validate cti",
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

func execute(ctx context.Context, baseDir string) error {
	slog.Info("Validating bundle", slog.String("path", baseDir))
	bd := bundle.New(baseDir)
	if err := bd.Read(); err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	// TODO: Validation for usage of indirect dependencies
	if err := bunman.Validate(bd); err != nil {
		return fmt.Errorf("validate bundle: %w", err)
	}
	slog.Info("No errors found")
	return nil
}
