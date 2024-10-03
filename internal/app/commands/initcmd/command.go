package initcmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/acronis/go-cti/internal/app/command"

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

func execute(ctx context.Context, baseDir string) error {
	return errors.New("not implemented")
}
